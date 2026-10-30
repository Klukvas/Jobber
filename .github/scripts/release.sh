#!/usr/bin/env bash
# Called only by the established deployment workflow. Never sources/prints .env.
# Releases one immutable SHA from the live clean schema (the source, at least MIN_SCHEMA) to the
# newest migration the release ships (the target). Migrations must stay additive: when the new
# backend fails after migrating, the previous backend is restarted on the target schema.
set -Eeuo pipefail
umask 077
PHASE=${1:?phase}; RELEASE=${2:?release}; BASE_HASH=${3:?base hash}; CADDY_HASH=${4:?Caddy hash}
[[ $RELEASE =~ ^[0-9a-f]{40}$ && $BASE_HASH =~ ^[0-9a-f]{64}$ && $CADDY_HASH =~ ^[0-9a-f]{64}$ ]] || exit 2
# Production has run AI credits (migration 052) since the manual creem-site release.
readonly MIN_SCHEMA=52
PROJECT="$HOME/jobber"
RELEASE_DIR="$PROJECT/releases/ci-$RELEASE"
BASE="$PROJECT/docker-compose.yml"
LIVE="$PROJECT/docker-compose.deploy.yml"
cd "$PROJECT"
fail() { printf 'release_error=%s\n' "$1" >&2; return 1; }
pg() { docker exec jobber-postgres sh -c 'export PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000"; exec psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "$1"' sh "$1" 2>/dev/null; }
schema() { local value; value=$(pg "SELECT version::text||':'||dirty::int FROM schema_migrations"); [[ $value =~ ^([0-9]+):0$ ]] && ((10#${BASH_REMATCH[1]} >= MIN_SCHEMA)) || { fail schema_not_clean_or_below_minimum; return 1; }; printf '%s' "$value"; }
compose() { docker compose --project-directory "$PROJECT" -f "$BASE" -f "$1" "${@:2}"; }
# Pulled images are pinned by repo digest; an image loaded by a manual release has none and is pinned by ID.
image_ref() { local id digest; id=$(docker inspect --format '{{.Image}}' "$1") || return 1; digest=$(docker image inspect --format '{{if .RepoDigests}}{{index .RepoDigests 0}}{{end}}' "$id") || return 1; printf '%s' "${digest:-$id}"; }
is_stopped() { [[ $(docker inspect --format '{{.State.Running}}' "$1") == false ]]; }
stopped_cleanly() { is_stopped "$1" && [[ $(docker inspect --format '{{.State.ExitCode}} {{.State.OOMKilled}}' "$1") == '0 false' ]]; }
http_code() { local raw; raw=$(docker exec jobber-backend wget -T 5 -S -O /dev/null "http://127.0.0.1:8080$1" 2>&1 || true); printf '%s\n' "$raw" | awk '/HTTP\/[0-9.]+ [0-9]+/ {print $2;exit}'; }
public_code() { curl --connect-timeout 3 --max-time 5 --silent --show-error -o /dev/null -w '%{http_code}' "$1"; }
public_ready() {
 local status
 for ((probe=0;probe<15;probe++)); do
  status=$(public_code https://cursopus.com/) || status=000
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
# Named objects must all exist; constraint counts a later additive migration may extend are lower bounds.
validate49() {
 [[ $(pg "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('interview_versions','interview_sessions','interview_answers','interview_receipts','interview_operations','ai_quota_reservations')") == 6 ]] || { fail missing_feature_tables; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND i.indisvalid AND i.indisready AND c.relname IN ('interview_versions_history','interview_sessions_history','interview_answers_owner','interview_operations_claim','interview_operations_lease','interview_operations_history','interview_operations_single_review','interview_operations_active_plan','ai_quota_reservations_owner_expiry')") == 9 ]] || { fail missing_feature_indexes; return 1; }
 [[ $(pg "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='interview_operations' AND column_name IN ('provider_started','charge_released','lease_token','lease_until','input','fingerprint')") == 6 ]] || { fail missing_operation_columns; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid IN ('interview_versions'::regclass,'interview_sessions'::regclass,'interview_answers'::regclass,'interview_receipts'::regclass,'interview_operations'::regclass,'ai_quota_reservations'::regclass) AND contype='f' AND convalidated") -ge 15 ]] || { fail missing_feature_foreign_keys; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_indexes WHERE indexname IN ('interview_operations_single_review','interview_operations_active_plan') AND indexdef LIKE 'CREATE UNIQUE INDEX%' AND indexdef LIKE '% WHERE %'") == 2 ]] || { fail missing_partial_uniqueness; return 1; }
}
validate50() {
 validate49 || return 1
 [[ $(pg "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='job_contacts' AND column_name IN ('id','job_id','name','role','email','telegram','linkedin','note','created_at','updated_at')") == 10 ]] || { fail missing_contact_columns; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid WHERE i.indrelid='job_contacts'::regclass AND i.indisvalid AND i.indisready AND c.relname IN ('job_contacts_pkey','job_contacts_parent')") == 2 ]] || { fail missing_contact_indexes; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid='job_contacts'::regclass AND contype='f' AND convalidated AND confrelid='jobs'::regclass AND confdeltype='c'") -ge 1 ]] || { fail missing_contact_parent; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid='job_contacts'::regclass AND contype='c' AND convalidated") -ge 6 ]] || { fail missing_contact_bounds; return 1; }
}
validate51() {
 validate50 || return 1
 [[ $(pg "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='resume_tailoring_operations' AND column_name IN ('user_id','kind','request_key','job_id','fingerprint','status','object_id','payload','error_code','provider_started','ai_receipt_id','deadline_at','created_at','updated_at')") == 14 ]] || { fail missing_tailoring_columns; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid WHERE i.indrelid='resume_tailoring_operations'::regclass AND i.indisvalid AND i.indisready AND c.relname IN ('resume_tailoring_operations_pkey','resume_tailoring_import_source')") == 2 ]] || { fail missing_tailoring_indexes; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid='resume_tailoring_operations'::regclass AND contype='f' AND convalidated AND confrelid='users'::regclass AND confdeltype='c'") -ge 1 ]] || { fail missing_tailoring_owner; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid='resume_tailoring_operations'::regclass AND contype='c' AND convalidated") -ge 2 ]] || { fail missing_tailoring_bounds; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_constraint WHERE conrelid='resume_tailoring_operations'::regclass AND contype='p' AND pg_get_constraintdef(oid) LIKE 'PRIMARY KEY (user_id, kind, request_key%'") == 1 ]] || { fail missing_tailoring_identity; return 1; }
}
validate52() {
 validate51 || return 1
 [[ $(pg "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('credit_grants','credit_payment_holds','ai_credit_operations','ai_provider_budget','ai_credit_attempts')") == 5 ]] || { fail missing_credit_tables; return 1; }
 [[ $(pg "SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid WHERE i.indrelid='credit_grants'::regclass AND i.indisvalid AND i.indisready AND i.indisunique AND c.relname IN ('credit_grants_free_once','credit_grants_transaction','credit_grants_period')") == 3 ]] || { fail missing_credit_uniqueness; return 1; }
}
# Fingerprints read only the columns that existed before migrating, so an added column is no change.
columns() { pg "SELECT string_agg(quote_ident(column_name),',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name='$1'"; }
fingerprint_sql() { printf "SET TIME ZONE 'UTC'; SELECT count(*)::text||':'||coalesce(md5(string_agg(md5(to_jsonb(t)::text),'' ORDER BY id)),'-') FROM (SELECT %s FROM %s) t" "$2" "$1"; }
fingerprint() { pg "$(fingerprint_sql "$1" "$2")"; }
preflight() {
 test -f .env && test -f "$BASE" && test -f "$LIVE" || { fail missing_project_files; return 1; }
 [[ $(sha256sum "$BASE" | cut -d' ' -f1) == "$BASE_HASH" ]] || { fail base_compose_drift; return 1; }
 [[ $(sha256sum Caddyfile | cut -d' ' -f1) == "$CADDY_HASH" ]] || { fail caddy_drift; return 1; }
 command -v jq >/dev/null || return 1; command -v flock >/dev/null || return 1
 [[ $(pg "SELECT current_setting('server_version_num')::int/10000") == 15 ]] || { fail expected_postgres15; return 1; }
 local state; state=$(schema) || return 1
 validate52 || return 1
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
TARGET_MIGRATIONS="$RELEASE_DIR/migrations"
test -d "$TARGET_MIGRATIONS"
TARGET=0
for migration in "$TARGET_MIGRATIONS"/*.sql; do
 name=${migration##*/}
 [[ $name =~ ^([0-9]{6})_.+\.(up|down)\.sql$ ]] || fail invalid_migration_filename
 version=$((10#${BASH_REMATCH[1]}))
 if ((version > TARGET)); then TARGET=$version;fi
done
((TARGET >= MIN_SCHEMA)) || fail release_below_minimum_schema
TARGET_PREFIX=$(printf '%06d' "$TARGET")
compgen -G "$TARGET_MIGRATIONS/${TARGET_PREFIX}_*.up.sql" >/dev/null && compgen -G "$TARGET_MIGRATIONS/${TARGET_PREFIX}_*.down.sql" >/dev/null || fail incomplete_target_migration
exec 9>"$PROJECT/.interview-release.lock"
flock -n 9 || fail concurrent_release
# A dropped SSH session must not kill the release with the entry closed: output is best effort.
trap '' PIPE HUP
say() { printf '%s\n' "$1" 2>/dev/null || true; }
preflight
OLD_BACKEND=$(image_ref jobber-backend); OLD_FRONTEND=$(image_ref jobber-frontend)
[[ $OLD_BACKEND =~ ^(.+@)?sha256:[0-9a-f]{64}$ && $OLD_FRONTEND =~ ^(.+@)?sha256:[0-9a-f]{64}$ ]] || fail missing_old_image_refs
ENV_HASH=$(sha256sum .env | cut -d' ' -f1)
SOURCE_SCHEMA=$(schema)
SOURCE=$((10#${SOURCE_SCHEMA%:0}))
((SOURCE <= TARGET)) || fail schema_ahead_of_release
compgen -G "$TARGET_MIGRATIONS/$(printf '%06d' "$SOURCE")_*.up.sql" >/dev/null || fail missing_source_migration
cp -p "$BASE" "$RELEASE_DIR/previous-base.yml"
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
# The live override must look exactly like what releases write: pinned images, flat quoted backend
# settings and an optional migrations mount. Anything else would be dropped, so stop before touching it.
bad_line=$(awk 'NR==1 {if ($0!="services:") {bad=NR; exit}; next}
 /^  (backend|frontend):$/ {svc=$1; block=""; next}
 svc!="" && /^    image: [^ ]+$/ {next}
 svc=="backend:" && /^    environment:$/ {block="env"; next}
 svc=="backend:" && /^    volumes:$/ {block="vol"; next}
 block=="env" && /^      [A-Z][A-Z0-9_]*: "[^"$`\\]*"$/ {next}
 block=="vol" && /^      - [^ ]+:\/app\/migrations:ro$/ {next}
 {bad=NR; exit}
 END {if (bad) {print bad; exit 1}}' "$LIVE") || fail "unsupported_live_override_line_${bad_line:-unreadable}"
# Backend settings kept only in the live override (such as ADMIN_EMAILS) carry over to every
# override this release writes. The payment flags never do: they come from the server's .env
# alone, so a pin left in an older live override is dropped here.
LIVE_ENV=$(awk '/^  (backend|frontend):$/ {svc=$1; env=0; next} svc=="backend:" && /^    environment:$/ {env=1; next} /^    [^ ]/ {env=0} env' "$LIVE" | grep -vE '^      FEATURE_(PAYMENTS|BILLING_WEBHOOK)_ENABLED: ' || true)
# Both print "<payments>|<webhook>" exactly as the backend receives them; empty means unset.
rendered_payment_flags() { compose "$1" config --format json 2>/dev/null | jq -er '.services.backend.environment | "\(.FEATURE_PAYMENTS_ENABLED // "")|\(.FEATURE_BILLING_WEBHOOK_ENABLED // "")"'; }
running_payment_flags() { docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' jobber-backend | awk '/^FEATURE_PAYMENTS_ENABLED=/ {sub(/^[^=]*=/, ""); payments=$0} /^FEATURE_BILLING_WEBHOOK_ENABLED=/ {sub(/^[^=]*=/, ""); webhook=$0} END {printf "%s|%s", payments, webhook}'; }
# Reads them the way config.getEnvAsBool does (payments default off, an empty webhook flag follows
# payments), but refuses any spelling other than true/false instead of falling back silently.
effective_payment_flags() {
 local payments webhook
 IFS='|' read -r payments webhook <<<"$1"
 case $payments in ''|false) payments=false;; true) ;; *) return 1;; esac
 case $webhook in '') webhook=$payments;; true|false) ;; *) return 1;; esac
 printf 'payments=%s webhook=%s' "$payments" "$webhook"
}
write_override() {
 cat >"$1" <<YAML
services:
  backend:
    image: $2
YAML
 [[ -z $LIVE_ENV ]] || printf '    environment:\n%s\n' "$LIVE_ENV" >>"$1"
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
# One rollback override per clean schema the release can stop at. Its migrations end at that
# schema, so the previous backend starts on whatever it finds and never advances it. A manual
# rollback later uses releases/ci-<sha>/rollback-<live schema>.yml the same way.
for ((version=SOURCE; version<=TARGET; version++)); do
 through="$RELEASE_DIR/migrations-through-$version"
 mkdir -p "$through"
 for migration in "$TARGET_MIGRATIONS"/*.sql; do
  name=${migration##*/}
  [[ $name =~ ^([0-9]{6}) ]] || fail invalid_migration_filename
  if ((10#${BASH_REMATCH[1]} <= version)); then cp "$migration" "$through/$name";fi
 done
 for migration in "$through"/*.sql; do
  [[ ${migration##*/} =~ ^([0-9]{6}) ]] && ((10#${BASH_REMATCH[1]} <= version)) || fail invalid_rollback_migrations
 done
 chmod -R a+rX "$through"
 write_override "$RELEASE_DIR/rollback-$version.yml" "$OLD_BACKEND" "$OLD_FRONTEND" "$through"
done
# A frontend-only release restarts the old backend on the source schema;
# a new backend uses its own complete migration source after verified backup.
SOURCE_MOUNT=
[[ -n $BACKEND ]] || SOURCE_MOUNT="$RELEASE_DIR/migrations-through-$SOURCE"
write_override "$RELEASE_DIR/new-images.yml" "$NEW_BACKEND" "$NEW_FRONTEND" "$SOURCE_MOUNT"
# A release never switches payments on or off: what the new override renders from .env must
# match the running backend, or nothing stops. Turning payments on is an operator step.
PLANNED_FLAGS=$(rendered_payment_flags "$RELEASE_DIR/new-images.yml") || fail payment_flags_unreadable
RUNNING_FLAGS=$(running_payment_flags) || fail payment_flags_unreadable
PAYMENTS=$(effective_payment_flags "$PLANNED_FLAGS") || fail invalid_payment_flags
CURRENT_PAYMENTS=$(effective_payment_flags "$RUNNING_FLAGS") || fail invalid_payment_flags
[[ $PAYMENTS == "$CURRENT_PAYMENTS" ]] || fail payment_flags_would_change
# A checkout UI must not go out over a backend without payments, nor a backend with payments
# behind a frontend that hides them. CI labels every frontend image with the value it built.
if [[ -n $FRONTEND ]]; then
 [[ payments=$(docker image inspect --format '{{index .Config.Labels "com.cursopus.feature-payments"}}' "$FRONTEND") == "${PAYMENTS%% *}" ]] || fail frontend_payments_mismatch
fi
ENTRY_CLOSED=false; COMMITTED=false; RESTORE_CONTAINER=
rollback() {
 local reason=$? current version
 # set -E also fires this trap inside $(...); only the main shell may roll back.
 ((BASH_SUBSHELL == 0)) || exit "${reason:-1}"
 trap - ERR
 set +e
 printf 'release_failed=1 details=sanitized\n' >&2
 [[ -z $RESTORE_CONTAINER ]] || { docker stop -t 10 "$RESTORE_CONTAINER"; docker rm -v "$RESTORE_CONTAINER"; } >/dev/null 2>&1
 if [[ $ENTRY_CLOSED == true && $COMMITTED == false ]]; then
  recovery_stop jobber-frontend 45 || { printf 'rollback=frontend_stop_failed entry=unconfirmed\n' >&2; exit 1; }
  recovery_stop jobber-backend 20 || { printf 'rollback=backend_stop_failed entry=unconfirmed\n' >&2; exit 1; }
  current=$(schema 2>/dev/null)
  version=${current%:0}
  # Only a clean schema between source and target is safe; structure checks never block recovery.
  if [[ $current =~ ^[0-9]+:0$ ]] && ((10#$version >= SOURCE && 10#$version <= TARGET)); then
   version=$((10#$version))
   compose "$RELEASE_DIR/rollback-$version.yml" up -d --no-deps backend >/dev/null 2>&1 || { printf 'rollback=backend_start_failed entry=closed\n' >&2; exit 1; }
  else
   printf 'rollback=blocked schema=dirty_unreadable_or_unexpected entry=closed\n' >&2
   exit 1
  fi
  local ready=false
  for ((attempt=0;attempt<60;attempt++)); do if health; then ready=true;break;fi;sleep 2;done
  if [[ $ready == true ]]; then
   cp "$RELEASE_DIR/rollback-$version.yml" "$LIVE" || exit 1
   compose "$LIVE" up -d --no-deps frontend >/dev/null 2>&1 || recovery_frontend_failure frontend_start_failed
   public_ready || recovery_frontend_failure frontend_unhealthy
   printf 'rollback=healthy schema=%s\n' "$current" >&2
  else printf 'rollback=unhealthy entry=closed\n' >&2;fi
 fi
 exit "${reason:-1}"
}
trap rollback ERR
# Written only now, so a release that stopped nothing never shadows the last real rollback target.
cp "$LIVE" "$RELEASE_DIR/previous-images.yml"
# nginx SIGQUIT drains existing proxy requests; its AI proxy timeout is 30s.
ENTRY_CLOSED=true
docker stop -t 45 jobber-frontend >/dev/null
stopped_cleanly jobber-frontend
docker stop -t 20 jobber-backend >/dev/null
stopped_cleanly jobber-backend
[[ $(docker ps --filter ancestor="$OLD_BACKEND" --format '{{.ID}}' | wc -l) -eq 0 ]] || fail old_backend_still_running
say 'drain=complete old_backend=stopped entry=closed'
SUBS_COLS=$(columns subscriptions); JOBS_COLS=$(columns jobs); RESUMES_COLS=$(columns resume_builders)
[[ -n $SUBS_COLS && -n $JOBS_COLS && -n $RESUMES_COLS ]] || fail missing_fingerprint_columns
BEFORE_SUBS=$(fingerprint subscriptions "$SUBS_COLS"); BEFORE_JOBS=$(fingerprint jobs "$JOBS_COLS"); BEFORE_RESUMES=$(fingerprint resume_builders "$RESUMES_COLS")
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP="$PROJECT/backups/ci-$RELEASE-$STAMP.sql.gz"
mkdir -p "$PROJECT/backups"
# Streamed straight into gzip, so no plaintext dump is ever left on disk.
# shellcheck disable=SC2016 # Expand credentials only inside the database container.
timeout 120 docker exec jobber-postgres sh -c 'exec pg_dump --no-owner --no-acl -U "$POSTGRES_USER" "$POSTGRES_DB"' | gzip -c >"$BACKUP"
gzip -t "$BACKUP"
[[ $(gzip -dc "$BACKUP" | wc -c) -ge 10000 ]] || fail backup_too_small
sha256sum "$BACKUP" >"$BACKUP.sha256"
sha256sum --check --status "$BACKUP.sha256"
# Restore into an isolated PG15 instance with no network or published ports.
RESTORE_CONTAINER="jobber-restore-${RELEASE:0:7}-$STAMP"
PG_IMAGE=$(docker inspect --format '{{.Image}}' jobber-postgres)
docker run -d --network none --name "$RESTORE_CONTAINER" -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_DB=restorecheck "$PG_IMAGE" >/dev/null
for ((attempt=0;attempt<30;attempt++)); do docker exec "$RESTORE_CONTAINER" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1 && break;sleep 1;done
gzip -dc "$BACKUP" | timeout 120 docker exec -i "$RESTORE_CONTAINER" env PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d restorecheck >/dev/null 2>&1
RESTORED=$(docker exec "$RESTORE_CONTAINER" env PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d restorecheck -tAc "SELECT version::text||':'||dirty::int FROM schema_migrations")
[[ $RESTORED == "$SOURCE_SCHEMA" ]] || fail restore_schema_mismatch
for table in subscriptions jobs resume_builders; do
 case $table in subscriptions) expected=$BEFORE_SUBS cols=$SUBS_COLS;;jobs) expected=$BEFORE_JOBS cols=$JOBS_COLS;;resume_builders) expected=$BEFORE_RESUMES cols=$RESUMES_COLS;;esac
 restored=$(docker exec "$RESTORE_CONTAINER" env PGCONNECT_TIMEOUT=5 PGOPTIONS="-c statement_timeout=15000 -c lock_timeout=5000" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d restorecheck -tAc "$(fingerprint_sql "$table" "$cols")")
 [[ $restored == "$expected" ]] || fail restore_data_mismatch
done
docker stop -t 10 "$RESTORE_CONTAINER" >/dev/null
# -v drops the restored copy of the database with the container.
docker rm -v "$RESTORE_CONTAINER" >/dev/null
RESTORE_CONTAINER=
say "backup=verified checksum=ok isolated_pg15_restore=ok schema=$RESTORED"
compose "$RELEASE_DIR/new-images.yml" up -d --no-deps backend >/dev/null
READY=false
for ((attempt=0;attempt<60;attempt++)); do if health;then READY=true;break;fi;sleep 2;done
[[ $READY == true ]] || fail new_backend_unhealthy
EXPECTED_SCHEMA="$TARGET:0"
[[ -n $BACKEND ]] || EXPECTED_SCHEMA=$SOURCE_SCHEMA
[[ $(schema) == "$EXPECTED_SCHEMA" ]] || fail target_schema_not_expected_clean
validate52
[[ $(running_payment_flags) == "$PLANNED_FLAGS" ]] || fail payment_flags_not_from_env
BACKEND_ENV=$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' jobber-backend)
while IFS= read -r line; do
 [[ -n $line ]] || continue
 key=${line#      }; key=${key%%:*}; value=${line#*: \"}; value=${value%\"}
 grep -qxF "$key=$value" <<<"$BACKEND_ENV" || fail live_override_env_not_carried
done <<<"$LIVE_ENV"
[[ $(docker inspect --format '{{.Image}}' jobber-backend) == "$(docker image inspect --format '{{.Id}}' "$NEW_BACKEND")" ]] || fail wrong_running_backend
# Every backend since the AI-credits release (match-score cache included) serves these behind auth.
JOB=00000000-0000-4000-8000-000000000000
ROUTES=(/api/v1/jobs "/api/v1/jobs/$JOB/interview-prep/operations/00000000-0000-4000-8000-000000000001" "/api/v1/jobs/$JOB/contacts" "/api/v1/jobs/$JOB/resume-tailoring/source/00000000-0000-4000-8000-000000000001" /api/v1/subscription /api/v1/match-score)
PUBLIC_ROUTE="/api/v1/jobs/$JOB/resume-tailoring/source/00000000-0000-4000-8000-000000000001"
for route in "${ROUTES[@]}"; do
 [[ $(http_code "$route") == 401 ]] || fail protected_route_not_401
done
[[ $(fingerprint subscriptions "$SUBS_COLS") == "$BEFORE_SUBS" && $(fingerprint jobs "$JOBS_COLS") == "$BEFORE_JOBS" && $(fingerprint resume_builders "$RESUMES_COLS") == "$BEFORE_RESUMES" ]] || fail existing_data_changed
[[ $(sha256sum .env | cut -d' ' -f1) == "$ENV_HASH" ]] || fail environment_changed
cp "$RELEASE_DIR/new-images.yml" "$LIVE"
compose "$LIVE" up -d --no-deps frontend >/dev/null
[[ $(docker inspect --format '{{.State.Running}}' jobber-frontend) == true ]] || fail frontend_not_running
[[ $(docker inspect --format '{{.Image}}' jobber-frontend) == "$(docker image inspect --format '{{.Id}}' "$NEW_FRONTEND")" ]] || fail wrong_running_frontend
public_ready || fail public_frontend_not_200
[[ $(public_code "https://cursopus.com$PUBLIC_ROUTE") == 401 ]] || fail public_api_not_401
[[ $(sha256sum "$BASE" | cut -d' ' -f1) == "$BASE_HASH" && $(sha256sum Caddyfile | cut -d' ' -f1) == "$CADDY_HASH" ]] || fail static_config_changed
COMMITTED=true
trap - ERR
say "release=healthy schema=$EXPECTED_SCHEMA protected_routes=401 existing_data=preserved env=unchanged $PAYMENTS"
say "backend_image=$NEW_BACKEND"; say "frontend_image=$NEW_FRONTEND"; say "backup_file=$BACKUP"
# Keeps the images just replaced as the rollback target; a cleanup failure never fails the release.
cleanup_images() {
 local cleanup="$RELEASE_DIR/server-cleanup.sh" backend_id frontend_id container
 # Restore checks of earlier releases left stopped containers holding a copy of the database.
 for container in $(docker ps -aq --filter name=jobber-restore- --filter status=exited); do docker rm -v "$container" >/dev/null; done
 [[ -f $cleanup ]] || return 0
 backend_id=$(docker image inspect --format '{{.Id}}' "$OLD_BACKEND") || return 1
 frontend_id=$(docker image inspect --format '{{.Id}}' "$OLD_FRONTEND") || return 1
 timeout 600 bash "$cleanup" --lock-held --keep "$backend_id" --keep "$frontend_id"
}
cleanup_images || say 'cleanup=failed release=unaffected'
