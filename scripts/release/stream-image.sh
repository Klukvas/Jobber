#!/usr/bin/env bash
# Streams a local Docker image to a file on the release server without writing the archive to the
# local disk (a backend archive is ~1.2 GB), checks the server copy, and prints its SHA-256.
#
# Usage: scripts/release/stream-image.sh <image:tag> <remote-file>
# Env:   RELEASE_HOST (required, user@server), RELEASE_SSH_KEY (default ~/.ssh/gha_deploy)
set -Eeuo pipefail
[[ $# == 2 ]] || { echo "usage: $0 <image:tag> <remote-file>" >&2; exit 64; }
readonly image=$1 remote=$2
readonly host=${RELEASE_HOST:?set RELEASE_HOST to user@server}
ssh_opts=(-i "${RELEASE_SSH_KEY:-$HOME/.ssh/gha_deploy}" -o IdentitiesOnly=yes -o BatchMode=yes)

sha_file=$(mktemp)
trap 'rm -f "$sha_file"' EXIT
docker save "$image" \
  | tee >(shasum -a 256 | cut -d' ' -f1 >"$sha_file") \
  | ssh "${ssh_opts[@]}" "$host" "umask 077; cat > '$remote'"
# The hashing process substitution can still be flushing when the pipeline returns.
for _ in $(seq 1 100); do [[ -s $sha_file ]] && break; sleep 0.1; done
local_sha=$(cat "$sha_file")
[[ $local_sha =~ ^[a-f0-9]{64}$ ]] || { echo 'STOP: local archive hash missing' >&2; exit 1; }

remote_sha=$(ssh "${ssh_opts[@]}" "$host" "sha256sum '$remote'" | cut -d' ' -f1)
[[ $remote_sha == "$local_sha" ]] || { echo "STOP: server copy differs (local $local_sha, server $remote_sha)" >&2; exit 1; }
echo "$local_sha"
