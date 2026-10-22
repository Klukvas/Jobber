#!/usr/bin/env bash
# Called only by the established deployment workflow. Never sources/prints .env.
set -Eeuo pipefail
umask 077
PHASE=${1:?phase}; RELEASE=${2:?release}; BASE_HASH=${3:?base hash}; CADDY_HASH=${4:?Caddy hash}
[[ $RELEASE =~ ^[0-9a-f]{40}$ && $BASE_HASH =~ ^[0-9a-f]{64}$ && $CADDY_HASH =~ ^[0-9a-f]{64}$ ]] || exit 2
PROJECT="$HOME/jobber"
RELEASE_DIR="$PROJECT/releases/interview-$RELEASE"
BASE="$PROJECT/docker-compose.yml"
LIVE="$PROJECT/docker-compose.deploy.yml"
cd "$PROJECT"
fail() { printf 'release_error=%s\n' "$1" >&2; return 1; }
pg() { docker exec jobber-postgres sh -c 'export PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000"; exec psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "$1"' sh "$1" 2>/dev/null; }
schema() { local value; value=$(pg "SELECT version::text||':'||dirty::int FROM schema_migrations"); [[ $value =~ ^(48|49):0$ ]] || { fail schema_not_clean_48_or_49; return 1; }; printf '%s' "$value"; }
compose() { docker compose --project-directory "$PROJECT" -f "$BASE" -f "$1" "${@:2}"; }
image_digest() { docker image inspect --format '{{index .RepoDigests 0}}' "$(docker inspect --format '{{.Image}}' "$1")"; }
is_stopped() { [[ $(docker inspect --format '{{.State.Running}}' "$1") == false ]]; }
stopped_cleanly() { is_stopped "$1" && [[ $(docker inspect --format '{{.State.ExitCode}} {{.State.OOMKilled}}' "$1") == '0 false' ]]; }
http_code() { local raw; raw=$(docker exec jobber-backend wget -T 5 -S -O /dev/null "http://127.0.0.1:8080$1" 2>&1 || true); printf '%s\n' "$raw" | awk '/HTTP\/[0-9.]+ [0-9]+/ {print $2;exit}'; }
public_code() { curl --connect-timeout 3 --max-time 5 --silent --show-error -o /dev/null -w '%{http_code}' "$1"; }
public_ready() {
 local status
 for ((probe=0;probe<15;probe++)); do
  status=$(public_code https://jobber-app.com/) || status=000
  [[ $status == 200 ]] && return 0
  sleep 1
 done
 return 1
}
recovery_stop() { docker stop -t "$2" "$1" >/dev/null 2>&1 && is_stopped "$1"; }
recovery_frontend_failure() {
 local code=$1
 if recovery_stop jobber-frontend 45; then printf 'rollback=%s entry=closed\n' "$code" >&2
 else printf 'rollback=%s entry=unconfirmed frontend_stop_failed=1\n' "$code" >&2;fi
 exit 1
}
health() {
 [[ $(http_code /health) == 200 ]] || return 1
 docker exec jobber-backend wget -T 5 -q -O - http://127.0.0.1:8080/health | jq -e '.status=="healthy" and .services.postgres=="up" and .services.redis=="up"' >/dev/null
}
validate49() {
 [[ $(pg "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('interview_versions','interview_sessions','interview_answers','interview_receipts','interview_operations','ai_quota_reservations')") == 6 ]] || { fail missing_feature_tables; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND i.indisvalid AND i.indisready AND c.relname IN ('interview_versions_history','interview_sessions_history','interview_answers_owner','interview_operations_claim','interview_operations_lease','interview_operations_history','interview_operations_single_review','interview_operations_active_plan','ai_quota_reservations_owner_expiry')") == 9 ]] || { fail missing_feature_indexes; return 1; }
 [[ $(pg "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='interview_operations' AND column_name IN ('provider_started','charge_released','lease_token','lease_until','input','fingerprint')") == 6 ]] || { fail missing_operation_columns; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid IN ('interview_versions'::regclass,'interview_sessions'::regclass,'interview_answers'::regclass,'interview_receipts'::regclass,'interview_operations'::regclass,'ai_quota_reservations'::regclass) AND contype='f' AND convalidated") == 15 ]] || { fail missing_feature_foreign_keys; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_indexes WHERE indexname IN ('interview_operations_single_review','interview_operations_active_plan') AND indexdef LIKE 'CREATE UNIQUE INDEX%' AND indexdef LIKE '% WHERE %'") == 2 ]] || { fail missing_partial_uniqueness; return 1; }
}
fingerprint() {
 pg "SET TIME ZONE 'UTC'; SELECT count(*)::text||':'||coalesce(md5(string_agg(md5(to_jsonb(t)::text),'' ORDER BY id)),'-') FROM $1 t"
}
preflight() {
 test -f .env && test -f "$BASE" && test -f "$LIVE" || { fail missing_project_files; return 1; }
 [[ $(sha256sum "$BASE" | cut -d' ' -f1) == "$BASE_HASH" ]] || { fail base_compose_drift; return 1; }
 [[ $(sha256sum Caddyfile | cut -d' ' -f1) == "$CADDY_HASH" ]] || { fail caddy_drift; return 1; }
 command -v jq >/dev/null || return 1; command -v flock >/dev/null || return 1
 [[ $(pg "SELECT current_setting('server_version_num')::int/10000") == 15 ]] || { fail expected_postgres15; return 1; }
 local state; state=$(schema) || return 1
 if [[ $state == 49:0 ]]; then validate49 || return 1; else
  [[ $(pg "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('interview_versions','interview_sessions','interview_answers','interview_receipts','interview_operations','ai_quota_reservations')") == 0 ]] || { fail partial_feature_schema; return 1; }
 fi
 [[ $(docker inspect --format '{{.Config.StopSignal}}' jobber-frontend) == SIGQUIT ]] || { fail unexpected_frontend_stop_signal; return 1; }
 [[ $(df -Pk . | awk 'NR==2{print $4}') -ge 2097152 ]] || { fail insufficient_disk; return 1; }
 health || { fail unhealthy_current_backend; return 1; }
 printf 'preflight_schema=%s health=ok config=unchanged\n' "$state"
}
if [[ $PHASE == preflight ]]; then preflight; exit 0; fi
[[ $PHASE == deploy ]] || exit 2
BACKEND=${5:-}; FRONTEND=${6:-}
valid_ref() { [[ $1 =~ ^docker\.io/[a-z0-9_-]+/jobber-(backend|frontend):[0-9a-f]{7,40}$ ]]; }
[[ -z $BACKEND ]] || valid_ref "$BACKEND" || exit 2
[[ -z $FRONTEND ]] || valid_ref "$FRONTEND" || exit 2
[[ -n $BACKEND || -n $FRONTEND ]] || { printf 'deploy=no_changes\n'; exit 0; }
test -d "$RELEASE_DIR/migrations" && test -f "$RELEASE_DIR/migrations/000049_interview_preparation.up.sql"
exec 9>"$PROJECT/.interview-release.lock"
flock -n 9 || fail concurrent_release
preflight
OLD_BACKEND=$(image_digest jobber-backend); OLD_FRONTEND=$(image_digest jobber-frontend)
[[ $OLD_BACKEND =~ @sha256:[0-9a-f]{64}$ && $OLD_FRONTEND =~ @sha256:[0-9a-f]{64}$ ]] || fail missing_old_image_digests
ENV_HASH=$(sha256sum .env | cut -d' ' -f1)
SOURCE_SCHEMA=$(schema)
cp -p "$BASE" "$RELEASE_DIR/previous-base.yml"
cp -p "$LIVE" "$RELEASE_DIR/previous-images.yml"
for container in jobber-backend jobber-frontend; do
 docker exec "$container" true
 [[ $(docker image inspect --format '{{.Architecture}}' "$(docker inspect --format '{{.Image}}' "$container")") == amd64 ]] || fail wrong_live_architecture
done
NEW_BACKEND=${BACKEND:-$OLD_BACKEND}; NEW_FRONTEND=${FRONTEND:-$OLD_FRONTEND}
export DOCKER_CONFIG="$RELEASE_DIR/docker-auth"
for image in "$BACKEND" "$FRONTEND"; do
 [[ -n $image ]] || continue
 docker pull "$image" >/dev/null
 [[ $(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$image") == "$RELEASE" ]] || fail wrong_image_revision
 [[ $(docker image inspect --format '{{.Architecture}}' "$image") == amd64 ]] || fail wrong_new_architecture
done
# Activation and rollback use pulled immutable digests, never mutable dev tags.
[[ -z $BACKEND ]] || NEW_BACKEND=$(docker image inspect --format '{{index .RepoDigests 0}}' "$BACKEND")
[[ -z $FRONTEND ]] || NEW_FRONTEND=$(docker image inspect --format '{{index .RepoDigests 0}}' "$FRONTEND")
write_override() {
 cat >"$1" <<YAML
services:
  backend:
    image: $2
    environment:
      FEATURE_PAYMENTS_ENABLED: "false"
      FEATURE_BILLING_WEBHOOK_ENABLED: "false"
YAML
 if [[ -n ${4:-} ]]; then
  cat >>"$1" <<YAML
    volumes:
      - $4:/app/migrations:ro
YAML
 fi
 cat >>"$1" <<YAML
  frontend:
    image: $3
YAML
}
# Preserve startup compatibility after an earlier application rollback at49.
# A frontend-only release still recreates the old backend during the drain.
SOURCE_MOUNT=
[[ $SOURCE_SCHEMA != 49:0 ]] || SOURCE_MOUNT="$RELEASE_DIR/migrations"
write_override "$RELEASE_DIR/new-images.yml" "$NEW_BACKEND" "$NEW_FRONTEND" "$SOURCE_MOUNT"
write_override "$RELEASE_DIR/rollback-images.yml" "$OLD_BACKEND" "$OLD_FRONTEND"
cat >"$RELEASE_DIR/rollback-49.yml" <<YAML
services:
  backend:
    volumes:
      - $RELEASE_DIR/migrations:/app/migrations:ro
YAML
ENTRY_CLOSED=false; COMMITTED=false; RESTORE_CONTAINER=
rollback() {
 local reason=$? current
 trap - ERR
 set +e
 printf 'release_failed=1 details=sanitized\n' >&2
 [[ -z $RESTORE_CONTAINER ]] || docker stop -t 10 "$RESTORE_CONTAINER" >/dev/null 2>&1
 if [[ $ENTRY_CLOSED == true && $COMMITTED == false ]]; then
  recovery_stop jobber-frontend 45 || { printf 'rollback=frontend_stop_failed entry=unconfirmed\n' >&2; exit 1; }
  recovery_stop jobber-backend 20 || { printf 'rollback=backend_stop_failed entry=unconfirmed\n' >&2; exit 1; }
  current=$(schema 2>/dev/null)
  if [[ $current == 48:0 ]]; then
   # Do not mount 49 at 48: old binary would apply it on backup failure.
   compose "$RELEASE_DIR/rollback-images.yml" up -d --no-deps backend >/dev/null 2>&1 || { printf 'rollback=backend_start_failed entry=closed\n' >&2; exit 1; }
  elif [[ $current == 49:0 ]] && validate49 >/dev/null 2>&1; then
   docker compose --project-directory "$PROJECT" -f "$BASE" -f "$RELEASE_DIR/rollback-images.yml" -f "$RELEASE_DIR/rollback-49.yml" up -d --no-deps backend >/dev/null 2>&1 || { printf 'rollback=backend_start_failed entry=closed\n' >&2; exit 1; }
  else
   printf 'rollback=blocked schema=dirty_unreadable_or_unexpected entry=closed\n' >&2
   exit 1
  fi
  local ready=false
  for ((attempt=0;attempt<60;attempt++)); do if health; then ready=true;break;fi;sleep 2;done
  if [[ $ready == true ]]; then
   if [[ $current == 49:0 ]]; then
    write_override "$LIVE" "$OLD_BACKEND" "$OLD_FRONTEND" "$RELEASE_DIR/migrations" || exit 1
   else cp "$RELEASE_DIR/rollback-images.yml" "$LIVE" || exit 1;fi
   compose "$LIVE" up -d --no-deps frontend >/dev/null 2>&1 || recovery_frontend_failure frontend_start_failed
   public_ready || recovery_frontend_failure frontend_unhealthy
   printf 'rollback=healthy schema=%s\n' "$current" >&2
  else printf 'rollback=unhealthy entry=closed\n' >&2;fi
 fi
 exit "${reason:-1}"
}
trap rollback ERR
# nginx SIGQUIT drains existing proxy requests; its AI proxy timeout is 30s.
ENTRY_CLOSED=true
docker stop -t 45 jobber-frontend >/dev/null
stopped_cleanly jobber-frontend
docker stop -t 20 jobber-backend >/dev/null
stopped_cleanly jobber-backend
[[ $(docker ps --filter ancestor="$OLD_BACKEND" --format '{{.ID}}' | wc -l) -eq 0 ]] || fail old_backend_still_running
printf 'drain=complete old_backend=stopped entry=closed\n'
BEFORE_SUBS=$(fingerprint subscriptions); BEFORE_JOBS=$(fingerprint jobs); BEFORE_RESUMES=$(fingerprint resume_builders)
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP="$PROJECT/backups/interview-$RELEASE-$STAMP.sql.gz"
mkdir -p "$PROJECT/backups"
RAW="$RELEASE_DIR/backup-$STAMP.sql"
# shellcheck disable=SC2016 # Expand credentials only inside the database container.
timeout 120 docker exec jobber-postgres sh -c 'exec pg_dump --no-owner --no-acl -U "$POSTGRES_USER" "$POSTGRES_DB"' >"$RAW"
[[ $(wc -c <"$RAW") -ge 10000 ]] || fail backup_too_small
gzip -c "$RAW" >"$BACKUP"
gzip -t "$BACKUP"
sha256sum "$BACKUP" >"$BACKUP.sha256"
sha256sum --check --status "$BACKUP.sha256"
# Restore into an isolated PG15 instance with no network or published ports.
RESTORE_CONTAINER="jobber-restore-${RELEASE:0:7}-$STAMP"
PG_IMAGE=$(docker inspect --format '{{.Image}}' jobber-postgres)
docker run -d --network none --name "$RESTORE_CONTAINER" -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_DB=restorecheck "$PG_IMAGE" >/dev/null
for ((attempt=0;attempt<30;attempt++)); do docker exec "$RESTORE_CONTAINER" pg_isready -U postgres >/dev/null 2>&1 && break;sleep 1;done
gzip -dc "$BACKUP" | timeout 120 docker exec -i "$RESTORE_CONTAINER" env PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d restorecheck >/dev/null 2>&1
RESTORED=$(docker exec "$RESTORE_CONTAINER" env PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d restorecheck -tAc "SELECT version::text||':'||dirty::int FROM schema_migrations")
[[ $RESTORED == "$SOURCE_SCHEMA" ]] || fail restore_schema_mismatch
for table in subscriptions jobs resume_builders; do
 case $table in subscriptions) expected=$BEFORE_SUBS;;jobs) expected=$BEFORE_JOBS;;resume_builders) expected=$BEFORE_RESUMES;;esac
 restored=$(docker exec "$RESTORE_CONTAINER" env PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d restorecheck -tAc "SET TIME ZONE 'UTC'; SELECT count(*)::text||':'||coalesce(md5(string_agg(md5(to_jsonb(t)::text),'' ORDER BY id)),'-') FROM $table t")
 [[ $restored == "$expected" ]] || fail restore_data_mismatch
done
docker stop -t 10 "$RESTORE_CONTAINER" >/dev/null
RESTORE_CONTAINER=
printf 'backup=verified checksum=ok isolated_pg15_restore=ok schema=%s\n' "$RESTORED"
compose "$RELEASE_DIR/new-images.yml" up -d --no-deps backend >/dev/null
READY=false
for ((attempt=0;attempt<60;attempt++)); do if health;then READY=true;break;fi;sleep 2;done
[[ $READY == true ]] || fail new_backend_unhealthy
[[ $(schema) == 49:0 ]] || fail target_schema_not_49_clean
validate49
docker exec jobber-backend sh -c 'test "$FEATURE_PAYMENTS_ENABLED" = false && test "$FEATURE_BILLING_WEBHOOK_ENABLED" = false'
[[ $(docker inspect --format '{{.Image}}' jobber-backend) == "$(docker image inspect --format '{{.Id}}' "$NEW_BACKEND")" ]] || fail wrong_running_backend
for route in /api/v1/jobs /api/v1/jobs/00000000-0000-4000-8000-000000000000/interview-prep /api/v1/jobs/00000000-0000-4000-8000-000000000000/interview-prep/operations/00000000-0000-4000-8000-000000000001; do
 [[ $(http_code "$route") == 401 ]] || fail protected_route_not_401
done
[[ $(fingerprint subscriptions) == "$BEFORE_SUBS" && $(fingerprint jobs) == "$BEFORE_JOBS" && $(fingerprint resume_builders) == "$BEFORE_RESUMES" ]] || fail existing_data_changed
[[ $(sha256sum .env | cut -d' ' -f1) == "$ENV_HASH" ]] || fail environment_changed
cp "$RELEASE_DIR/new-images.yml" "$LIVE"
compose "$LIVE" up -d --no-deps frontend >/dev/null
[[ $(docker inspect --format '{{.State.Running}}' jobber-frontend) == true ]] || fail frontend_not_running
[[ $(docker inspect --format '{{.Image}}' jobber-frontend) == "$(docker image inspect --format '{{.Id}}' "$NEW_FRONTEND")" ]] || fail wrong_running_frontend
public_ready || fail public_frontend_not_200
[[ $(public_code https://jobber-app.com/api/v1/jobs/00000000-0000-4000-8000-000000000000/interview-prep) == 401 ]] || fail public_api_not_401
[[ $(sha256sum "$BASE" | cut -d' ' -f1) == "$BASE_HASH" && $(sha256sum Caddyfile | cut -d' ' -f1) == "$CADDY_HASH" ]] || fail static_config_changed
COMMITTED=true
trap - ERR
printf 'release=healthy schema=49:0 protected_routes=401 existing_data=preserved env=unchanged\n'
printf 'backend_image=%s\nfrontend_image=%s\nbackup_file=%s\n' "$NEW_BACKEND" "$NEW_FRONTEND" "$BACKUP"
