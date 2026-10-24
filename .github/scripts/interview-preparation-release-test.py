#!/usr/bin/env python3
"""Hermetic failure/recovery checks for the real release shell; no remote writes."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

HELPER = Path(__file__).with_name('interview-preparation-release.sh').resolve()
SHA = 'a' * 40
DIGEST = 'sha256:' + 'b' * 64

def contact_probes(calls):
    return [call for call in calls
            if (call[0] == 'curl' or 'wget' in call)
            and call[-1].startswith(('http://', 'https://'))
            and call[-1].endswith('/contacts')]

MOCK = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name=pathlib.Path(sys.argv[0]).name; a=sys.argv[1:]
p=pathlib.Path(os.environ['MOCK_STATE']); s=json.loads(p.read_text())
def done(out='',code=0):
 p.write_text(json.dumps(s)); print(out,end=''); sys.exit(code)
s['calls'].append([name]+a)
case=s['case']
if name in ('sleep','flock'): done()
if name=='timeout':
 os.execvp(a[1], a[1:])
if name=='df': done('Filesystem 1024-blocks Used Available Capacity Mounted\nx 9999999 0 9999999 0% /\n')
if name=='curl':
 if '/api/' in a[-1]: done('404' if '/contacts' in a[-1] and not s['supports_contacts'] else '401')
 key='recovery_http' if s.get('rolling_back') else 'activation_http'
 s[key]=s.get(key,0)+1
 if case=='delayed_readiness' and s[key]<=2: done('513')
 if not s.get('rolling_back') and case in ('activation_redirect','rollback_stop_frontend','rollback_stop_backend'): done('302')
 if s.get('rolling_back') and case in ('recovery_redirect','recovery_http_failure'): done('302' if case=='recovery_redirect' else '510')
 done('200')
if name=='docker':
 if a[0]=='pull': done()
 if a[0]=='ps': done()
 if a[0]=='stop':
  if s.get('new_started') and not s.get('rolling_back'):
   if case=='rollback_stop_frontend' and a[-1]=='jobber-frontend': done(code=1)
   if case=='rollback_stop_backend' and a[-1]=='jobber-backend': done(code=1)
  if 'restore' not in a[-1]: s[a[-1]+'Running']=False
  done()
 if a[0]=='run': done('restore-id\n')
 if a[0]=='image':
  f=a[a.index('--format')+1]
  done(('amd64' if 'Architecture' in f else ('a'*40 if 'revision' in f else ('image-id' if '.Id' in f else 'docker.io/x/jobber-backend@sha256:'+'b'*64)))+'\n')
 if a[0]=='inspect':
  f=a[a.index('--format')+1]; c=a[-1]
  if 'StopSignal' in f: done('SIGQUIT\n')
  if 'Running' in f: done(str(s.get(c+'Running',True)).lower()+'\n')
  if 'ExitCode' in f: done(('137 false' if case=='forced_drain' else '0 false')+'\n')
  done('image-id\n')
 if a[0]=='compose':
  target=a[-1]; rollback=any('rollback' in x for x in a)
  if target=='backend':
   files=[pathlib.Path(a[i+1]) for i,x in enumerate(a) if x=='-f']
   mount=None
   for file in files:
    for line in file.read_text().splitlines():
     if ':/app/migrations:ro' in line: mount=pathlib.Path(line.strip()[2:].split(':/app/')[0])
   source=max(int(p.name[:6]) for p in mount.glob('*.up.sql')) if mount else 51
   s.setdefault('backend_starts',[]).append({'rollback':rollback,'source':source,'schema_before':s['schema'],'backup_verified':s.get('backup_verified',False)})
   if rollback:
    s['rolling_back']=True
    if case=='rollback_backend_failure': done(code=1)
   else:
    if case=='frontend_only_after_rollback': s['retained51mount']=source==51
    if case=='frontend_only50': s['retained50mount']=source==50
    s['new_started']=True
   s['supports_contacts']=False if rollback else (s['activating_contact_backend'] or s['supports_contacts'])
   s['schema']='51:1' if not rollback and case=='migration_dirty' else str(source)+':0'
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
    code='200' if a[-1].endswith('/health') else ('404' if '/contacts' in a[-1] and not s['supports_contacts'] else '401')
    if code=='401' and s.get('new_started') and not s.get('rolling_back') and case in ('postcheck_failure','rollback_frontend_failure','recovery_redirect','recovery_http_failure'): code='510'
    done('  HTTP/1.1 '+code+' OK\n')
   done('{"status":"healthy","services":{"postgres":"up","redis":"up"}}')
  if 'server_version_num' in q: done('15\n')
  if 'schema_migrations' in q:
   if case=='schema_sql_failure': done(code=1)
   done(s['schema']+'\n')
  if 'information_schema.tables' in q:
   if "table_name='resume_tailoring_operations'" in q: done(('1' if s['schema'].startswith('51') or case=='partial_tailoring' else '0')+'\n')
   if "table_name='job_contacts'" in q: done(('1' if s['schema'].startswith('51') else '0')+'\n')
   done('6\n')
  if 'pg_index i' in q and "'resume_tailoring_operations'::regclass" in q: done(('1' if case=='missing_tailoring_index' else '2')+'\n')
  if 'pg_index i' in q: done(('2' if "'job_contacts'::regclass" in q else ('8' if case=='missing_index' else '9'))+'\n')
  if 'information_schema.columns' in q and "table_name='resume_tailoring_operations'" in q: done(('13' if case=='missing_tailoring_column' else '14')+'\n')
  if 'information_schema.columns' in q: done(('10' if "table_name='job_contacts'" in q else '6')+'\n')
  if 'pg_constraint' in q:
   if "'resume_tailoring_operations'::regclass" in q:
    if "contype='f'" in q: done(('0' if case=='missing_tailoring_fk' else '1')+'\n')
    if "contype='p'" in q: done(('0' if case=='missing_tailoring_pk' else '1')+'\n')
    done(('1' if case=='missing_tailoring_check' else '2')+'\n')
   if "'job_contacts'::regclass" in q: done(('6' if "contype='c'" in q else '1')+'\n')
   done(('14' if case=='missing_fk' else '15')+'\n')
  if 'CREATE UNIQUE INDEX' in q: done('2\n')
  if 'string_agg' in q: done('0:-\n')
  done()
done(code=1)
'''

class ReleaseProtocol(unittest.TestCase):
    def run_case(self, case, schema='50:0', phase='deploy'):
        with tempfile.TemporaryDirectory(prefix='contacts-release-') as directory:
            root = Path(directory)
            project = root / 'jobber'
            release = project / 'releases' / ('contacts-' + SHA)
            (release / 'migrations').mkdir(parents=True)
            for version, name in [(49, 'interview_preparation'), (50, 'job_contacts'), (51, 'resume_tailoring_operations')]:
                for direction in ('up', 'down'):
                    (release / f'migrations/{version:06}_{name}.{direction}.sql').write_text('-- fixture')
            for name in ('.env', 'docker-compose.yml', 'docker-compose.deploy.yml', 'Caddyfile'):
                (project / name).write_text('synthetic fixture\n')
            mockbin = root / 'bin'
            mockbin.mkdir()
            for name in ('docker', 'curl', 'sleep', 'df', 'flock', 'timeout'):
                path = mockbin / name
                path.write_text(MOCK)
                path.chmod(0o755)
            state = root / 'state.json'
            state.write_text(json.dumps({'case': 'postcheck_failure' if case=='frontend_only_after_rollback' else case, 'schema': schema, 'source': schema, 'supports_contacts': False, 'activating_contact_backend': case != 'frontend_only50', 'calls': []}))
            digest = hashlib.sha256(b'synthetic fixture\n').hexdigest()
            env = dict(os.environ, HOME=str(root), PATH=str(mockbin)+os.pathsep+os.environ['PATH'], MOCK_STATE=str(state))
            command = ['bash', str(HELPER), phase, SHA, digest, digest,
                       'docker.io/x/jobber-backend:' + SHA[:7],
                       'docker.io/x/jobber-frontend:' + SHA[:7]]
            if case == 'frontend_only50': command[-2] = ''
            result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=20)
            if case == 'frontend_only_after_rollback':
                self.assertIn('rollback=healthy schema=51:0', result.stderr)
                self.assertIn(':/app/migrations:ro', (project / 'docker-compose.deploy.yml').read_text())
                current = json.loads(state.read_text())
                current.update(case=case, source='51:0', rolling_back=False, new_started=False, activating_contact_backend=False, calls=[])
                state.write_text(json.dumps(current))
                command[-2] = ''
                result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=20)
            return result, json.loads(state.read_text())

    def test_clean_rollout_and_rerun(self):
        for schema in ('50:0', '51:0'):
            with self.subTest(schema=schema):
                result, state = self.run_case('success', schema)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn('release=healthy schema=51:0', result.stdout)
                self.assertTrue(state['supports_contacts'])
                self.assertEqual(len(contact_probes(state['calls'])), 1)
                tailoring=[c for c in state['calls'] if (c[0]=='curl' or 'wget' in c) and '/resume-tailoring/source/' in c[-1]]
                self.assertEqual(len(tailoring),2)
                calls = state['calls']
                starts = [i for i, call in enumerate(calls) if call[0:2] == ['docker', 'compose']]
                self.assertEqual([calls[i][-1] for i in starts], ['backend', 'frontend'])

    def test_bad_schema_and_missing_constraints_fail_before_stop(self):
        for case, schema in [('invalid_schema','48:0'), ('schema_sql_failure','50:0'),
                             ('missing_fk','50:0'), ('missing_index','50:0')]:
            with self.subTest(case=case):
                result, state = self.run_case(case, schema)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call[:2] == ['docker','stop'] for call in state['calls']))

    def test_partial_or_incomplete_tailoring_schema_fails_before_stop(self):
        for case,schema in [('partial_tailoring','50:0'), ('missing_tailoring_index','51:0'),
                            ('missing_tailoring_column','51:0'), ('missing_tailoring_fk','51:0'),
                            ('missing_tailoring_check','51:0'), ('missing_tailoring_pk','51:0'),
                            ('old_schema','49:0'), ('future_schema','52:0'), ('dirty_schema','51:1')]:
            with self.subTest(case=case):
                result,state=self.run_case(case,schema)
                self.assertNotEqual(result.returncode,0)
                self.assertFalse(any(call[:2]==['docker','stop'] for call in state['calls']))

    def test_backup_failure_restores50_without51_mount(self):
        result, state = self.run_case('backup_failure')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=50:0', result.stderr)
        rollback = [call for call in state['calls'] if 'compose' in call]
        self.assertFalse(any('rollback-51.yml' in ' '.join(call) for call in rollback))
        self.assertFalse(state.get('new_started', False))

    def test_frontend_only_at50_never_applies51(self):
        result, state = self.run_case('frontend_only50')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['schema'], '50:0')
        self.assertTrue(state['retained50mount'])
        self.assertIn('release=healthy schema=50:0', result.stdout)

    def test_prebackup_failure_uses_only50_source(self):
        for case in ('backup_failure', 'restore_failure', 'forced_drain'):
            with self.subTest(case=case):
                result, state = self.run_case(case)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(state['schema'], '50:0')
                for start in state.get('backend_starts', []):
                    self.assertTrue(start['rollback'])
                    self.assertEqual(start['source'], 50)

    def test_backup_failure_at51_retains_full51_source(self):
        result, state = self.run_case('backup_failure', '51:0')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=51:0', result.stderr)
        self.assertEqual(state['backend_starts'][0]['source'], 51)

    def test_new_backend_only_starts_after_restore_verification(self):
        result, state = self.run_case('success')
        self.assertEqual(result.returncode, 0, result.stderr)
        start = state['backend_starts'][0]
        self.assertEqual(start['schema_before'], '50:0')
        self.assertTrue(start['backup_verified'])
        self.assertEqual(start['source'], 51)

    def test_dirty_migration_stays_closed(self):
        result, state = self.run_case('migration_dirty')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=blocked', result.stderr)
        self.assertFalse(state.get('jobber-frontendRunning', True))

    def test_forced_drain_blocks_migration(self):
        result, state = self.run_case('forced_drain')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(state.get('new_started', False))
        self.assertFalse(any('pg_dump' in ' '.join(call) for call in state['calls']))

    def test_postmigration_failure_restores_old_image_with51_source(self):
        result, state = self.run_case('postcheck_failure')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=51:0', result.stderr)
        self.assertTrue(any('rollback-51.yml' in ' '.join(call) for call in state['calls']))

    def test_restore_failure_recovers50_before_new_start(self):
        result, state = self.run_case('restore_failure')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=healthy schema=50:0', result.stderr)
        self.assertFalse(state.get('new_started', False))

    def test_rollback_frontend_failure_never_claims_healthy(self):
        result, state = self.run_case('rollback_frontend_failure')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=frontend_start_failed entry=closed', result.stderr)
        self.assertFalse(state['jobber-frontendRunning'])
        self.assertNotIn('rollback=healthy', result.stderr)

    def test_recovery_http_failure_or_redirect_closes_started_frontend(self):
        for case in ('recovery_redirect', 'recovery_http_failure'):
            with self.subTest(case=case):
                result, state = self.run_case(case)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('rollback=frontend_unhealthy entry=closed', result.stderr)
                self.assertFalse(state['jobber-frontendRunning'])
                self.assertNotIn('rollback=healthy', result.stderr)

    def test_failed_recovery_stops_never_start_old_backend(self):
        for case in ('rollback_stop_frontend', 'rollback_stop_backend'):
            with self.subTest(case=case):
                result, state = self.run_case(case)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('entry=unconfirmed', result.stderr)
                self.assertNotIn('entry=closed', result.stderr)
                self.assertNotIn('rollback=healthy', result.stderr)
                self.assertFalse(state.get('rolling_back', False))
                failed = 'jobber-frontend' if case.endswith('frontend') else 'jobber-backend'
                self.assertTrue(state[failed + 'Running'])

    def test_activation_redirect_is_not_success(self):
        result, state = self.run_case('activation_redirect')
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('release=healthy', result.stdout)
        self.assertIn('rollback=healthy schema=51:0', result.stderr)
        self.assertEqual(state['activation_http'], 15)

    def test_exact200_readiness_can_arrive_late(self):
        result, state = self.run_case('delayed_readiness')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(state['activation_http'], 3)
        self.assertIn('release=healthy', result.stdout)

    def test_frontend_only_after_full51_rollback_retains_source_mount(self):
        result, state = self.run_case('frontend_only_after_rollback')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(state['retained51mount'])
        self.assertEqual(state['schema'], '51:0')
        self.assertFalse(state['supports_contacts'])
        self.assertEqual(contact_probes(state['calls']), [])
        pulls = [call for call in state['calls'] if call[:2] == ['docker','pull']]
        self.assertEqual(len(pulls), 1)
        self.assertIn('jobber-frontend', pulls[0][-1])

    def test_rollback_start_failure_never_claims_healthy(self):
        result, _ = self.run_case('rollback_backend_failure')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('rollback=backend_start_failed', result.stderr)
        self.assertNotIn('rollback=healthy', result.stderr)

if __name__ == '__main__':
    unittest.main(verbosity=2)
