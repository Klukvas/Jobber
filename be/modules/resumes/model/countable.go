package model

// CountableResumeCondition is the SQL that separates a resume from a
// placeholder row that never became one.
//
// Until the upload flow moved the INSERT into finalization, `POST
// /resumes/upload-url` wrote the row itself: `storage_type = 's3'`,
// `is_active = false`, no `file_url`, and no object behind it yet. Every
// abandoned upload — a closed tab, a rejected file, a dropped connection — left
// one of those behind permanently. They were never resumes, but they were
// counted like resumes, so a free plan's three slots could be filled entirely
// by uploads that never happened, and nothing but a support request could free
// them.
//
// The four clauses are what the presign-era writer produced and what nothing
// else can produce:
//
//   - `storage_type = 's3'` — an external-URL resume is complete on creation.
//   - `is_active = false` — finalization writes the row active, so no upload
//     this service completes can match.
//   - `file_url IS NULL` — an uploaded resume has no external URL either way;
//     this rules out a row that carries one.
//   - `updated_at = created_at` — the row has never been touched since it was
//     written. This is what keeps a *real* upload that the customer later
//     switched off from being mistaken for a ghost: switching it off is an
//     update, and an update moves `updated_at`.
//
// Three places have to agree on this, because they gate the same limit from
// different sides: the subscription service counts with it, the resume
// repository re-counts with it inside the transaction that writes, and the same
// transaction uses its negation to recognise a placeholder it is allowed to
// claim. They share these constants so they cannot drift apart and start
// refusing uploads the count would have allowed — or claiming a row that is
// somebody's real resume.
const UnfinalizedUploadCondition = `(
		storage_type = 's3'
		AND is_active = false
		AND file_url IS NULL
		AND updated_at = created_at
	)`

// CountableResumeCondition is exactly "not a placeholder".
const CountableResumeCondition = `NOT ` + UnfinalizedUploadCondition

// IsUnfinalizedUpload is CountableResumeCondition applied to a row already in
// memory — the same four facts, in Go.
//
// Used where the SQL cannot be: the idempotent finalize path reads the row by
// id and has to decide whether it represents a completed upload. A presign-era
// placeholder answered "yes" there, so a client finalizing an id that never had
// an object behind it was told its upload had succeeded and handed a resume
// whose download endpoint pointed at nothing.
func (r *Resume) IsUnfinalizedUpload() bool {
	return r.StorageType == StorageTypeS3 &&
		!r.IsActive &&
		r.FileURL == nil &&
		r.UpdatedAt.Equal(r.CreatedAt)
}
