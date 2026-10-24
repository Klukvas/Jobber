-- Additive workflow receipts. Resource IDs are provenance, deliberately without
-- foreign keys: document/job deletion must not erase usage or permit replays.
CREATE TABLE resume_tailoring_operations (
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind text NOT NULL CHECK (kind IN ('import','recognition','generate','apply')),
 request_key uuid NOT NULL,
 job_id uuid NOT NULL,
 fingerprint text NOT NULL,
 status text NOT NULL CHECK (status IN ('pending','parsed','succeeded','failed','interrupted')),
 object_id uuid,
 payload jsonb,
 error_code text NOT NULL DEFAULT '',
 provider_started boolean NOT NULL DEFAULT false,
 ai_receipt_id uuid,
 deadline_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (user_id,kind,request_key)
);
CREATE INDEX resume_tailoring_import_source ON resume_tailoring_operations(user_id,object_id) WHERE kind='import';
