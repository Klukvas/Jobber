#!/usr/bin/env bash
# Frees disk on the build machine after a release upload: stale Docker build cache (unused for a week)
# and dangling images. Tagged images stay, because the next thin release builds FROM the current one.
#
# Usage: scripts/release/local-cleanup.sh
set -Eeuo pipefail
free_gb() { df -Pk "$HOME" | awk 'NR==2{printf "%.1f", $4/1048576}'; }
before=$(free_gb)
docker builder prune -f --filter until=168h >/dev/null
docker image prune -f >/dev/null
echo "LOCAL CLEANUP free ${before} GB -> $(free_gb) GB"
