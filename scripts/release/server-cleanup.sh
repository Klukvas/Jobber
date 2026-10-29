#!/usr/bin/env bash
# Frees disk on the production server after a successful release. Removes the uploaded image archives
# (their images are already in the Docker store), the Docker build cache, and every image that no
# container uses and no release can roll back to. Always keeps:
#   - the images of all containers, running or stopped;
#   - the images referenced by the live compose override;
#   - the rollback target: images in the newest releases/*/compose.deploy.before.yml, plus any --keep.
# Only the latest release stays rollback-able; older rollback-release.sh scripts lose their images.
#
# Usage (root, on the server):
#   bash server-cleanup.sh [--dry-run] [--lock-held] [--keep sha256:<id>]...
#   --dry-run    print what would be removed, change nothing
#   --lock-held  the caller (activate-release.sh) already holds the release lock
set -Eeuo pipefail
readonly project=/root/jobber
readonly releases=$project/releases

dry_run=0
lock_held=0
declare -A keep=()
while [[ $# -gt 0 ]]; do
  case $1 in
    --dry-run) dry_run=1 ;;
    --lock-held) lock_held=1 ;;
    --keep)
      [[ ${2:-} =~ ^sha256:[a-f0-9]{64}$ ]] || { echo "STOP: --keep needs a sha256 image ID" >&2; exit 64; }
      keep[$2]=1; shift ;;
    *) echo "STOP: unknown argument $1" >&2; exit 64 ;;
  esac
  shift
done
[[ $(id -u) == 0 ]] || { echo 'STOP: run as root' >&2; exit 64; }
if [[ $lock_held == 0 ]]; then
  exec 9>"$project/.interview-release.lock"
  flock -w 10 9 || { echo 'STOP: another release owns the lock' >&2; exit 75; }
fi

# Dry-run lines go to stderr so the callers' >/dev/null does not hide them.
run() { if [[ $dry_run == 1 ]]; then echo "DRY-RUN $*" >&2; else "$@"; fi; }
free_gb() { df -Pk "$project" | awk 'NR==2{printf "%.1f", $4/1048576}'; }
before=$(free_gb)

# Captured first: a failing `docker ps` inside the for-list would not trip errexit.
containers=$(docker ps -aq)
for container in $containers; do keep[$(docker inspect -f '{{.Image}}' "$container")]=1; done
rollback_override=$(ls -t "$releases"/*/compose.deploy.before.yml 2>/dev/null | head -1 || true)
for file in "$project/docker-compose.deploy.yml" ${rollback_override:+"$rollback_override"}; do
  for id in $(grep -oE 'sha256:[a-f0-9]{64}' "$file" || true); do keep[$id]=1; done
done
[[ ${#keep[@]} -gt 0 ]] || { echo 'STOP: empty keep list, refusing to prune' >&2; exit 1; }
echo "keeping ${#keep[@]} images (containers, live override, rollback target ${rollback_override:-none})"

# One `docker image rm` per image: all its tags at once, or the bare ID when it has none.
declare -A refs=()
while read -r id ref; do
  [[ -n ${keep[$id]:-} ]] && continue
  [[ $ref == '<none>:<none>' ]] && ref=''
  refs[$id]+="${ref:+$ref }"
done < <(docker image ls --no-trunc --format '{{.ID}} {{.Repository}}:{{.Tag}}')

removed=0
failed=0
for id in "${!refs[@]}"; do
  read -ra targets <<<"${refs[$id]:-$id}"
  if run docker image rm "${targets[@]}" >/dev/null; then
    removed=$((removed + 1))
  else
    echo "WARN could not remove $id (${targets[*]})" >&2
    failed=$((failed + 1))
  fi
done

for archive in "$releases"/*/*-image.tar; do
  if [[ -e $archive ]]; then run rm -f "$archive"; fi
done
# No `docker image prune`: it ignores the keep list; the loop above already removed unkept untagged images.
run docker builder prune -af >/dev/null

echo "CLEANUP removed $removed images ($failed failed), image archives and build cache; free ${before} GB -> $(free_gb) GB"
