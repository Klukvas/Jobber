#!/usr/bin/env python3
"""Hermetic failure/recovery checks for the real release shell; no remote writes."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

HELPER = Path(__file__).with_name('release.sh').resolve()
SHA = 'a' * 40
OLD_ID = 'sha256:' + 'c' * 64
NEW_ID = 'sha256:' + 'd' * 64
MIGRATIONS = [(49, 'interview_preparation'), (50, 'job_contacts'), (51, 'resume_tailoring_operations'),
              (52, 'subscription_credits'), (53, 'synthetic_next'), (54, 'synthetic_after')]

def route_probes(calls, suffix):
    return [call for call in calls
            if (call[0] == 'curl' or 'wget' in call)
            and call[-1].startswith(('http://', 'https://'))
            and call[-1].endswith(suffix)]

MOCK = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name=pathlib.Path(sys.argv[0]).name; a=sys.argv[1:]
p=pathlib.Path(os.environ['MOCK_STATE']); s=json.loads(p.read_text())
def done(out='',code=0):
 p.write_text(json.dumps(s)); print(out,end=''); sys.exit(code)
s['calls'].append([name]+a)
case=s['case']
old_id='sha256:'+'c'*64; new_id='sha256:'+'d'*64
old_digest='docker.io/x/jobber-backend@sha256:'+'b'*64; new_digest='docker.io/x/jobber-backend@sha256:'+'e'*64
def is_new(ref): return 'e'*64 in ref or ref.endswith(':'+'a'*7)
def started_image(files, service):
 image=None
 for file in files:
  current=None
  for line in file.read_text().splitlines():
   if line.startswith('  ') and not line.startswith('   ') and line.strip().endswith(':'): current=line.strip()[:-1]
   if current==service and line.strip().startswith('image: '): image=line.strip()[7:]
 return image
if name in ('sleep','flock'): done()
if name=='timeout':
 os.execvp(a[1], a[1:])
if name=='df': done('Filesystem 1024-blocks Used Available Capacity Mounted\nx 9999999 0 9999999 0% /\n')
if name=='curl':
 if '/api/' in a[-1]: done('401')
 key='recovery_http' if s.get('rolling_back') else 'activation_http'
 s[key]=s.get(key,0)+1
 if case=='delayed_readiness' and s[key]<=2: done('513')
 if not s.get('rolling_back') and case in ('activation_redirect','rollback_stop_frontend','rollback_stop_backend'): done('302')
 if s.get('rolling_back') and case in ('recovery_redirect','recovery_http_failure'): done('302' if case=='recovery_redirect' else '510')
 done('200')
if name=='docker':
 if a[0]=='pull': done()
 if a[0]=='ps': done('old-restore-1\n' if 'name=jobber-restore-' in a else '')
 if a[0]=='stop':
  if s.get('new_started') and not s.get('rolling_back'):
   if case=='rollback_stop_frontend' and a[-1]=='jobber-frontend': done(code=1)
   if case=='rollback_stop_backend' and a[-1]=='jobber-backend': done(code=1)
  if 'restore' not in a[-1]: s[a[-1]+'Running']=False
  done()
 if a[0]=='run': done('restore-id\n')
 if a[0]=='image':
  f=a[a.index('--format')+1]
  if 'Architecture' in f: done('amd64\n')
  if 'revision' in f: done('a'*40+'\n')
  if '.Id' in f: done((new_id if is_new(a[-1]) else old_id)+'\n')
  # Images loaded by a manual release carry no repo digest; pulled ones always do.
  if case.startswith('local_old_images') and a[-1]==old_id: done('\n')
  done((new_digest if is_new(a[-1]) else old_digest)+'\n')
 if a[0]=='inspect':
  f=a[a.index('--format')+1]; c=a[-1]
  if 'StopSignal' in f: done('SIGQUIT\n')
  if 'Config.Env' in f: done('' if case=='env_not_carried' else 'ADMIN_EMAILS=admin@example.test\nFEATURE_PAYMENTS_ENABLED=false\n')
  if 'Running' in f: done(str(s.get(c+'Running',True)).lower()+'\n')
  if 'ExitCode' in f: done(('137 false' if case=='forced_drain' else '0 false')+'\n')
  done(s.get(c+'Image',old_id)+'\n')
 if a[0]=='rm': done()
 if a[0]=='compose':
  target=a[-1]; rollback=any('rollback' in x for x in a)
  files=[pathlib.Path(a[i+1]) for i,x in enumerate(a) if x=='-f']
  image=started_image(files, target)
  if image: s['jobber-'+target+'Image']=new_id if is_new(image) else old_id
  if target=='backend':
   mount=None
   for file in files:
    for line in file.read_text().splitlines():
     if ':/app/migrations:ro' in line: mount=pathlib.Path(line.strip()[2:].split(':/app/')[0])
   source=max(int(p.name[:6]) for p in mount.glob('*.up.sql')) if mount else s['target']
   s.setdefault('backend_starts',[]).append({'rollback':rollback,'source':source,'schema_before':s['schema'],'backup_verified':s.get('backup_verified',False)})
   if rollback:
    s['rolling_back']=True
    if case=='rollback_backend_failure': done(code=1)
   else:
    s['new_started']=True
    s['new_mount']=source
   s['schema']=str(s['target'])+':1' if not rollback and case=='migration_dirty' else str(source)+':0'
   if not rollback and case=='intermediate_failure': s['schema']=str(s['target']-1)+':0'
   s['jobber-backendRunning']=True
  if target=='frontend':
   s['jobber-frontendRunning']=True
   if s.get('rolling_back') and case=='rollback_frontend_failure': done(code=1)
  done()
 if a[0]=='exec':
  q=a[-1]
  if 'pg_dump' in ' '.join(a):
   if case in ('backup_failure','rollback_backend_failure'): done(code=1)
   done('-- synthetic backup\n'*1000)
  if 'restorecheck' in a:
   if case=='restore_failure': done(code=1)
   if 'string_agg' in q: done('0:-\n')
   if '-tAc' in a: s['backup_verified']=True; done(s['source']+'\n')
   sys.stdin.read() if '-i' in a else None
   done()
  if 'wget' in a:
   if '-S' in a:
    code='200' if a[-1].endswith('/health') else '401'
    if code=='200' and case=='intermediate_failure' and s.get('new_started') and not s.get('rolling_back'): code='500'
    if code=='401' and s.get('new_started') and not s.get('rolling_back') and case in ('postcheck_failure','local_old_images_postcheck','rollback_frontend_failure','recovery_redirect','recovery_http_failure'): code='510'
    done('  HTTP/1.1 '+code+' OK\n')
   done('{"status":"healthy","services":{"postgres":"up","redis":"up"}}')
  if 'server_version_num' in q: done('15\n')
  if 'schema_migrations' in q:
   if case=='schema_sql_failure': done(code=1)
   done(s['schema']+'\n')
  if 'information_schema.tables' in q:
   if "'credit_grants'" in q: done(('4' if case=='missing_credit_table' else '5')+'\n')
   done('6\n')
  if 'pg_index i' in q and "'credit_grants'::regclass" in q: done(('2' if case=='missing_credit_index' else '3')+'\n')
  if 'pg_index i' in q and "'resume_tailoring_operations'::regclass" in q: done(('1' if case=='missing_tailoring_index' else '2')+'\n')
  if 'pg_index i' in q: done(('2' if "'job_contacts'::regclass" in q else ('8' if case=='missing_index' else '9'))+'\n')
  if 'information_schema.columns' in q and "table_name='resume_tailoring_operations'" in q: done(('13' if case=='missing_tailoring_column' else '14')+'\n')
  if 'information_schema.columns' in q: done(('10' if "table_name='job_contacts'" in q else '6')+'\n')
  if 'pg_constraint' in q:
   if "'resume_tailoring_operations'::regclass" in q:
    if "contype='f'" in q: done(('0' if case=='missing_tailoring_fk' else '1')+'\n')
    if "contype='p'" in q: done(('0' if case=='missing_tailoring_pk' else '1')+'\n')
    done(('1' if case=='missing_tailoring_check' else '2')+'\n')
   if "'job_contacts'::regclass" in q:
    if "contype='c'" in q: done(('7' if case=='post_migration_extra_check' and s.get('new_started') else '6')+'\n')
    done('1\n')
   done(('14' if case=='missing_fk' else '15')+'\n')
  if 'CREATE UNIQUE INDEX' in q: done('2\n')
  if 'quote_ident' in q: done('id,user_id\n')
  if 'string_agg' in q:
   # A whole-row fingerprint changes once a migration adds a column; a fixed column list does not.
   if case=='column_added' and s.get('new_started') and '(SELECT ' not in q: done('1:changed\n')
   done('0:-\n')
  done()
done(code=1)
'''

CLEANUP = '''#!/usr/bin/env bash
printf '%s\\n' "$@" >"$HOME/cleanup-args"
exit {code}
'''

class ReleaseProtocol(unittest.TestCase):
    def run_case(self, case, schema='52:0', target=52, phase='deploy', frontend_only=False, backend_only=False, live_env=None, live_override=None):
        with tempfile.TemporaryDirectory(prefix='ci-release-') as directory:
            root = Path(directory)
            project = root / 'jobber'
            release = project / 'releases' / ('ci-' + SHA)
            (release / 'migrations').mkdir(parents=True)
            for version, name in MIGRATIONS:
                if version > target:
                    continue
                for direction in ('up', 'down'):
                    (release / f'migrations/{version:06}_{name}.{direction}.sql').write_text('-- fixture')
            cleanup = release / 'server-cleanup.sh'
            cleanup.write_text(CLEANUP.format(code=1 if case == 'cleanup_failure' else 0))
            for name in ('.env', 'docker-compose.yml', 'Caddyfile'):
                (project / name).write_text('synthetic fixture\n')
            (project / 'docker-compose.deploy.yml').write_text(live_override if live_override is not None else
                'services:\n  backend:\n    image: ' + OLD_ID + '\n    environment:\n' + (live_env + '\n' if live_env else '') +
                '      FEATURE_PAYMENTS_ENABLED: "false"\n      FEATURE_BILLING_WEBHOOK_ENABLED: "false"\n'
                '  frontend:\n    image: ' + OLD_ID + '\n')
            mockbin = root / 'bin'
            mockbin.mkdir()
            for name in ('docker', 'curl', 'sleep', 'df', 'flock', 'timeout'):
                path = mockbin / name
                path.write_text(MOCK)
                path.chmod(0o755)
            state = root / 'state.json'
            state.write_text(json.dumps({'case': 'postcheck_failure' if case == 'frontend_only_after_rollback' else case,
                                         'schema': schema, 'source': schema, 'target': target, 'calls': []}))
            digest = hashlib.sha256(b'synthetic fixture\n').hexdigest()
            env = dict(os.environ, HOME=str(root), PATH=str(mockbin)+os.pathsep+os.environ['PATH'], MOCK_STATE=str(state))
            command = ['bash', str(HELPER), phase, SHA, digest, digest,
                       'docker.io/x/jobber-backend:' + SHA[:7],
                       'docker.io/x/jobber-frontend:' + SHA[:7]]
            if frontend_only:
                command[-2] = ''
            if backend_only:
                command[-1] = ''
            result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=20)
            if case == 'frontend_only_after_rollback':
                self.assertIn(f'rollback=healthy schema={target}:0', result.stderr)
                self.assertIn(':/app/migrations:ro', (project / 'docker-compose.deploy.yml').read_text())
                current = json.loads(state.read_text())
                current.update(case=case, source=f'{target}:0', rolling_back=False, new_started=False, calls=[])
                state.write_text(json.dumps(current))
                command[-2] = ''
                result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=20)
            args = root / 'cleanup-args'
            final = json.loads(state.read_text())
            final['cleanup_args'] = args.read_text().split() if args.exists() else None
            final['live_override'] = (project / 'docker-compose.deploy.yml').read_text()
            source_rollback = release / f"rollback-{int(schema.split(':')[0])}.yml"
            final['rollback_override'] = source_rollback.read_text() if source_rollback.exists() else ''
            final['raw_backups'] = [path.name for path in release.glob('backup-*.sql')]
            return result, final

    def test_clean_rollout_to_same_or_next_schema(self):
        for target in (52, 53):
            with self.subTest(target=target):
                result, state = self.run_case('success', '52:0', target)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(f'release=healthy schema={target}:0', result.stdout)
                for suffix in ('/contacts', '/subscription', '/match-score'):
                    self.assertEqual(len(route_probes(state['calls'], suffix)), 1, suffix)
                tailoring = [c for c in state['calls'] if (c[0] == 'curl' or 'wget' in c) and '/resume-tailoring/source/' in c[-1]]
                self.assertEqual(len(tailoring), 2)
                calls = state['calls']
                starts = [i for i, call in enumerate(calls) if call[0:2] == ['docker', 'compose']]
                self.assertEqual([calls[i][-1] for i in starts], ['backend', 'frontend'])

    def test_bad_schema_and_missing_constraints_fail_before_stop(self):
        for case, schema in [('invalid_schema', '51:0'), ('old_schema', '49:0'), ('dirty_schema', '52:1'),
                             ('schema_sql_failure', '52:0'), ('missing_fk', '52:0'), ('missing_index', '52:0'),
                             ('missing_tailoring_index', '52:0'), ('missing_tailoring_column', '52:0'),
                             ('missing_tailoring_fk', '52:0'), ('missing_tailoring_check', '52:0'),
                             ('missing_tailoring_pk', '52:0'), ('missing_credit_table', '52:0'),
                             ('missing_credit_index', '52:0')]:
            with self.subTest(case=case):
                result, state = self.run_case(case, schema)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call[:2] == ['docker', 'stop'] for call in state['calls']))

    def test_schema_ahead_of_release_fails_before_stop(self):
        result, state = self.run_case('success', '53:0', 52)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('release_error=schema_ahead_of_release', result.stderr)
        self.assertFalse(any(call[:2] == ['docker', 'stop'] for call in state['calls']))

    def test_preflight_phase_reports_live_schema(self):
        result, state = self.run_case('success', '52:0', phase='preflight')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('preflight_schema=52:0 health=ok config=unchanged', result.stdout)
        self.assertFalse(any(call[:2] == ['docker', 'stop'] for call in state['calls']))

    def test_manually_loaded_old_images_are_pinned_by_id(self):
        result, state = self.run_case('local_old_images', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('release=healthy schema=53:0', result.stdout)
        self.assertIn('image: ' + OLD_ID, state['rollback_override'])
        self.assertNotIn('@sha256:', state['rollback_override'])

    def test_rollback_to_manually_loaded_images_uses_their_id(self):
        result, state = self.run_case('local_old_images_postcheck', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=53:0', result.stderr)
        self.assertIn('image: ' + OLD_ID, state['live_override'])
        self.assertNotIn('@sha256:', state['live_override'])
        self.assertEqual(state['jobber-backendImage'], OLD_ID)

    def test_backup_failure_restores_source_without_target_mount(self):
        result, state = self.run_case('backup_failure', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=52:0', result.stderr)
        self.assertEqual(state['backend_starts'][-1], {'rollback': True, 'source': 52, 'schema_before': '52:0', 'backup_verified': False})
        self.assertFalse(state.get('new_started', False))

    def test_frontend_only_never_applies_target(self):
        result, state = self.run_case('success', '52:0', 53, frontend_only=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['schema'], '52:0')
        self.assertEqual(state['new_mount'], 52)
        self.assertIn('release=healthy schema=52:0', result.stdout)

    def test_prebackup_failure_uses_only_source(self):
        for case in ('backup_failure', 'restore_failure', 'forced_drain'):
            with self.subTest(case=case):
                result, state = self.run_case(case, '52:0', 53)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(state['schema'], '52:0')
                for start in state.get('backend_starts', []):
                    self.assertTrue(start['rollback'])
                    self.assertEqual(start['source'], 52)

    def test_new_backend_only_starts_after_restore_verification(self):
        result, state = self.run_case('success', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        start = state['backend_starts'][0]
        self.assertEqual(start['schema_before'], '52:0')
        self.assertTrue(start['backup_verified'])
        self.assertEqual(start['source'], 53)

    def test_dirty_migration_stays_closed(self):
        result, state = self.run_case('migration_dirty', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=blocked', result.stderr)
        self.assertFalse(state.get('jobber-frontendRunning', True))

    def test_forced_drain_blocks_migration(self):
        result, state = self.run_case('forced_drain', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(state.get('new_started', False))
        self.assertFalse(any('pg_dump' in ' '.join(call) for call in state['calls']))

    def test_postmigration_failure_restores_old_image_with_target_source(self):
        result, state = self.run_case('postcheck_failure', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=53:0', result.stderr)
        last = state['backend_starts'][-1]
        self.assertTrue(last['rollback'])
        self.assertEqual(last['source'], 53)
        self.assertIn('migrations-through-53:/app/migrations:ro', state['live_override'])
        self.assertEqual(state['jobber-backendImage'], OLD_ID)

    def test_postcheck_failure_without_new_migration_restores_source(self):
        result, state = self.run_case('postcheck_failure', '52:0', 52)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=52:0', result.stderr)
        self.assertEqual(state['backend_starts'][-1]['source'], 52)

    def test_restore_failure_recovers_source_before_new_start(self):
        result, state = self.run_case('restore_failure', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=52:0', result.stderr)
        self.assertFalse(state.get('new_started', False))

    def test_rollback_frontend_failure_never_claims_healthy(self):
        result, state = self.run_case('rollback_frontend_failure', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=frontend_start_failed entry=closed', result.stderr)
        self.assertFalse(state['jobber-frontendRunning'])
        self.assertNotIn('rollback=healthy', result.stderr)

    def test_recovery_http_failure_or_redirect_closes_started_frontend(self):
        for case in ('recovery_redirect', 'recovery_http_failure'):
            with self.subTest(case=case):
                result, state = self.run_case(case, '52:0', 53)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('rollback=frontend_unhealthy entry=closed', result.stderr)
                self.assertFalse(state['jobber-frontendRunning'])
                self.assertNotIn('rollback=healthy', result.stderr)

    def test_failed_recovery_stops_never_start_old_backend(self):
        for case in ('rollback_stop_frontend', 'rollback_stop_backend'):
            with self.subTest(case=case):
                result, state = self.run_case(case, '52:0', 53)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('entry=unconfirmed', result.stderr)
                self.assertNotIn('entry=closed', result.stderr)
                self.assertNotIn('rollback=healthy', result.stderr)
                self.assertFalse(state.get('rolling_back', False))
                failed = 'jobber-frontend' if case.endswith('frontend') else 'jobber-backend'
                self.assertTrue(state[failed + 'Running'])

    def test_activation_redirect_is_not_success(self):
        result, state = self.run_case('activation_redirect', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('release=healthy', result.stdout)
        self.assertIn('rollback=healthy schema=53:0', result.stderr)
        self.assertEqual(state['activation_http'], 15)

    def test_exact200_readiness_can_arrive_late(self):
        result, state = self.run_case('delayed_readiness', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['activation_http'], 3)
        self.assertIn('release=healthy', result.stdout)

    def test_frontend_only_after_target_rollback_retains_source_mount(self):
        result, state = self.run_case('frontend_only_after_rollback', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['new_mount'], 53)
        self.assertEqual(state['schema'], '53:0')
        pulls = [call for call in state['calls'] if call[:2] == ['docker', 'pull']]
        self.assertEqual(len(pulls), 1)
        self.assertIn('jobber-frontend', pulls[0][-1])

    def test_rollback_start_failure_never_claims_healthy(self):
        result, _ = self.run_case('rollback_backend_failure', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=backend_start_failed', result.stderr)
        self.assertNotIn('rollback=healthy', result.stderr)

    def test_cleanup_keeps_replaced_images_after_success(self):
        result, state = self.run_case('success', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['cleanup_args'], ['--lock-held', '--keep', OLD_ID, '--keep', OLD_ID])
        self.assertEqual(state['jobber-backendImage'], NEW_ID)
        self.assertEqual(state['jobber-frontendImage'], NEW_ID)

    def test_cleanup_never_runs_after_failure(self):
        result, state = self.run_case('postcheck_failure', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertIsNone(state['cleanup_args'])

    def test_cleanup_failure_never_fails_release(self):
        result, state = self.run_case('cleanup_failure', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('release=healthy schema=53:0', result.stdout)
        self.assertIn('cleanup=failed release=unaffected', result.stdout)

    def test_intermediate_clean_schema_rolls_back_on_that_schema(self):
        result, state = self.run_case('intermediate_failure', '52:0', 54)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('release_error=new_backend_unhealthy', result.stderr)
        self.assertIn('rollback=healthy schema=53:0', result.stderr)
        self.assertEqual(state['backend_starts'][-1]['source'], 53)
        self.assertEqual(state['schema'], '53:0')

    def test_added_constraint_after_migration_is_not_a_failure(self):
        result, state = self.run_case('post_migration_extra_check', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('release=healthy schema=53:0', result.stdout)

    def test_live_schema_above_minimum(self):
        for frontend_only in (False, True):
            with self.subTest(frontend_only=frontend_only):
                result, state = self.run_case('success', '53:0', 54, frontend_only=frontend_only)
                self.assertEqual(result.returncode, 0, result.stderr)
                expected = 53 if frontend_only else 54
                self.assertIn(f'release=healthy schema={expected}:0', result.stdout)
                self.assertEqual(state['new_mount'], expected)

    def test_backend_only_release_keeps_running_frontend_image(self):
        result, state = self.run_case('success', '52:0', 53, backend_only=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        pulls = [call for call in state['calls'] if call[:2] == ['docker', 'pull']]
        self.assertEqual(len(pulls), 1)
        self.assertIn('jobber-backend', pulls[0][-1])
        self.assertEqual(state['jobber-backendImage'], NEW_ID)
        self.assertEqual(state['jobber-frontendImage'], OLD_ID)

    def test_failure_inside_command_substitution_rolls_back_once(self):
        result, state = self.run_case('migration_dirty', '52:0', 53)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stderr.count('release_failed=1'), 1, result.stderr)

    def test_backup_artifacts_are_not_left_behind(self):
        result, state = self.run_case('success', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['raw_backups'], [])
        removed = [call[-1] for call in state['calls'] if call[:3] == ['docker', 'rm', '-v']]
        self.assertEqual(len(removed), 2)
        self.assertTrue(removed[0].startswith('jobber-restore-' + SHA[:7]))
        self.assertEqual(removed[1], 'old-restore-1')

    def test_live_override_env_carries_over(self):
        line = '      ADMIN_EMAILS: "admin@example.test"'
        result, state = self.run_case('success', '52:0', 53, live_env=line)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(line, state['live_override'])
        self.assertIn(line, state['rollback_override'])
        self.assertEqual(state['live_override'].count('FEATURE_PAYMENTS_ENABLED'), 1)

    def test_unsafe_live_override_env_fails_before_stop(self):
        for line in ('      ADMIN_EMAILS: "$(id)"', '      ADMIN_EMAILS: unquoted'):
            with self.subTest(line=line):
                result, state = self.run_case('success', '52:0', 53, live_env=line)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('release_error=unsupported_live_override_line_5', result.stderr)
                self.assertNotIn('id)', result.stderr)
                self.assertFalse(any(call[:2] == ['docker', 'stop'] for call in state['calls']))

    def test_override_shapes_a_release_would_lose_fail_before_stop(self):
        head = 'services:\n  backend:\n    image: ' + OLD_ID + '\n    environment:\n'
        tail = '  frontend:\n    image: ' + OLD_ID + '\n'
        for name, override in [
                ('blank_line_in_env', head + '\n      ADMIN_EMAILS: "a@b.c"\n' + tail),
                ('inline_map', 'services:\n  backend:\n    image: x\n    environment: {A: "b"}\n' + tail),
                ('extra_key', head + '      A: "b"\n    restart: always\n' + tail),
                ('frontend_env', head + '      A: "b"\n' + tail + '    environment:\n      B: "c"\n'),
                ('other_service', head + '      A: "b"\n' + tail + '  my-svc:\n    image: y\n'),
                ('not_compose', 'synthetic fixture\n')]:
            with self.subTest(name=name):
                result, state = self.run_case('success', '52:0', 53, live_override=override)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('release_error=unsupported_live_override_line_', result.stderr)
                self.assertFalse(any(call[:2] == ['docker', 'stop'] for call in state['calls']))

    def test_live_override_with_payments_on_fails_before_stop(self):
        override = ('services:\n  backend:\n    image: ' + OLD_ID + '\n    environment:\n'
                    '      FEATURE_PAYMENTS_ENABLED: "true"\n      FEATURE_BILLING_WEBHOOK_ENABLED: "false"\n'
                    '  frontend:\n    image: ' + OLD_ID + '\n')
        result, state = self.run_case('success', '52:0', 53, live_override=override)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('release_error=live_override_enables_payments', result.stderr)
        self.assertFalse(any(call[:2] == ['docker', 'stop'] for call in state['calls']))

    def test_ci_written_override_is_accepted_as_live(self):
        override = ('services:\n  backend:\n    image: docker.io/x/jobber-backend@sha256:' + 'b' * 64 + '\n'
                    '    environment:\n      FEATURE_PAYMENTS_ENABLED: "false"\n      FEATURE_BILLING_WEBHOOK_ENABLED: "false"\n'
                    '    volumes:\n      - /root/jobber/releases/ci-x/migrations-through-52:/app/migrations:ro\n'
                    '  frontend:\n    image: docker.io/x/jobber-frontend@sha256:' + 'b' * 64 + '\n')
        result, state = self.run_case('success', '52:0', 53, live_override=override)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_added_column_is_not_a_data_change(self):
        result, state = self.run_case('column_added', '52:0', 53)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('release=healthy schema=53:0', result.stdout)

    def test_env_missing_from_new_backend_rolls_back(self):
        result, state = self.run_case('env_not_carried', '52:0', 53, live_env='      ADMIN_EMAILS: "admin@example.test"')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('release_error=live_override_env_not_carried', result.stderr)
        self.assertIn('rollback=healthy schema=53:0', result.stderr)

if __name__ == '__main__':
    unittest.main(verbosity=2)
