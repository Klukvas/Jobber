-- Additive only: no rewrites or constraints on existing user data.
CREATE TABLE interview_versions (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 job_id uuid NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 source_resume_id uuid REFERENCES resume_builders(id) ON DELETE SET NULL,
 provenance_resume_id uuid NOT NULL, source_hash text NOT NULL,
 snapshot jsonb NOT NULL, plan jsonb NOT NULL,
 studied text[] NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX interview_versions_history ON interview_versions(user_id,job_id,created_at DESC,id);
CREATE TABLE interview_sessions (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 job_id uuid NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 version_id uuid NOT NULL REFERENCES interview_versions(id) ON DELETE CASCADE,
 status text NOT NULL DEFAULT 'active' CHECK(status IN ('active','paused','completed')),
 revision integer NOT NULL DEFAULT 0, questions jsonb NOT NULL,
 summary jsonb, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX interview_sessions_history ON interview_sessions(user_id,job_id,created_at DESC,id);
CREATE TABLE interview_answers (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 session_id uuid NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
 question_id text NOT NULL, answer_text text NOT NULL CHECK(length(answer_text)<=12000),
 skipped boolean NOT NULL, feedback jsonb, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(session_id,question_id)
);
CREATE INDEX interview_answers_owner ON interview_answers(user_id,session_id,created_at,id);
CREATE TABLE interview_receipts (
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, action text NOT NULL,
 request_key uuid NOT NULL, fingerprint text NOT NULL, object_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(user_id,action,request_key)
);
CREATE TABLE interview_operations (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 job_id uuid NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 kind text NOT NULL CHECK(kind IN ('plan','feedback','summary')),
 business_key text NOT NULL, fingerprint text NOT NULL, input jsonb NOT NULL,
 status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','succeeded','failed','interrupted')),
 error_code text NOT NULL DEFAULT '', provider_started boolean NOT NULL DEFAULT false, charge_released boolean NOT NULL DEFAULT false, attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 3),
 lease_until timestamptz, lease_token uuid,
 version_id uuid REFERENCES interview_versions(id) ON DELETE CASCADE,
 session_id uuid REFERENCES interview_sessions(id) ON DELETE CASCADE,
 answer_id uuid REFERENCES interview_answers(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX interview_operations_claim ON interview_operations(created_at,id) WHERE status='queued';
CREATE INDEX interview_operations_lease ON interview_operations(lease_until) WHERE status='running';
CREATE INDEX interview_operations_history ON interview_operations(user_id,job_id,created_at DESC,id);

CREATE UNIQUE INDEX interview_operations_single_review ON interview_operations(user_id,kind,business_key) WHERE kind!='plan';
CREATE UNIQUE INDEX interview_operations_active_plan ON interview_operations(user_id,kind,business_key) WHERE kind='plan' AND status IN ('queued','running');

-- Temporary legacy requests reserve capacity without permanent usage on crash.
CREATE TABLE ai_quota_reservations (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL
);
CREATE INDEX ai_quota_reservations_owner_expiry ON ai_quota_reservations(user_id,expires_at);
