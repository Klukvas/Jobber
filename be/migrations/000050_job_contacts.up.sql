-- Additive: existing Job and user rows are unchanged.
CREATE TABLE job_contacts (
    id uuid PRIMARY KEY,
    job_id uuid NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    role text NOT NULL DEFAULT '' CHECK (char_length(role) <= 120),
    email text NOT NULL DEFAULT '' CHECK (char_length(email) <= 254),
    telegram text NOT NULL DEFAULT '' CHECK (char_length(telegram) <= 512),
    linkedin text NOT NULL DEFAULT '' CHECK (char_length(linkedin) <= 512),
    note text NOT NULL DEFAULT '' CHECK (char_length(note) <= 2000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX job_contacts_parent ON job_contacts(job_id, created_at, id);
