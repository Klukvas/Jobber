#!/usr/bin/env bash
# Compares the Cloudflare ranges trusted in Caddyfile with the ones Cloudflare
# publishes. A difference fails the run, and that failure is the prompt to update
# the list: see "Client IP behind Cloudflare" in README.md.
set -euo pipefail

CADDYFILE=${1:-Caddyfile}
PUBLISHED_URL=https://api.cloudflare.com/client/v4/ips

give_up() {
  echo "::error::$1"
  exit 2
}

listed=$(sed -n '/trusted_proxies static/,/client_ip_headers/p' "$CADDYFILE" | grep -oE '[0-9a-fA-F:.]+/[0-9]+' | sort) ||
  give_up "No trusted_proxies ranges found in $CADDYFILE"
# jq -e fails on an unsuccessful or empty reply, so an outage is never read as "in step".
published=$(curl -fsS --retry 3 --max-time 30 "$PUBLISHED_URL" | jq -er 'select(.success == true) | .result | .ipv4_cidrs + .ipv6_cidrs | .[]' | sort) ||
  give_up "Could not read the ranges from $PUBLISHED_URL"

if ! diff <(printf '%s\n' "$listed") <(printf '%s\n' "$published"); then
  echo "::error file=$CADDYFILE::Cloudflare ranges differ (< only in $CADDYFILE, > only at Cloudflare)"
  exit 1
fi

echo "$CADDYFILE trusts exactly the $(wc -l <<<"$published" | tr -d ' ') ranges Cloudflare publishes."
