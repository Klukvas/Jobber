package interviewprep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sourceSection struct{ key, table, columns string }

var sourceSections = []sourceSection{
	{"summary", "resume_summaries", "content"},
	{"experience", "resume_experiences", "company,position,location,start_date,end_date,is_current::text,description"},
	{"education", "resume_educations", "institution,degree,field_of_study,start_date,end_date,gpa,description"},
	{"skills", "resume_skills", "name,level"}, {"languages", "resume_languages", "name,proficiency"},
	{"certifications", "resume_certifications", "name,issuer,issue_date,expiry_date"},
	{"projects", "resume_projects", "name,start_date,end_date,description"},
	{"volunteering", "resume_volunteering", "organization,role,start_date,end_date,description"},
	{"custom_sections", "resume_custom_sections", "title,content"},
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func digest(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func validText(text string, max int, required bool) bool {
	return utf8.ValidString(text) && !strings.ContainsRune(text, 0) && utf8.RuneCountInString(text) <= max && (!required || strings.TrimSpace(text) != "")
}

// All source reads share one MVCC snapshot. Only visible content columns enter
// facts; contact details, IDs of hidden sections and layout metadata never do.
func (s *Store) Snapshot(ctx context.Context, owner, job, resume, locale string) (Snapshot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Snapshot{}, err
	}
	defer tx.Rollback(ctx)
	result, err := readSnapshot(ctx, tx, owner, job, resume, locale)
	if err != nil {
		return Snapshot{}, err
	}
	return result, tx.Commit(ctx)
}
func readSnapshot(ctx context.Context, tx pgx.Tx, owner, job, resume, locale string) (Snapshot, error) {
	result := Snapshot{JobID: job, ResumeID: resume, Locale: locale, Facts: []Fact{}}
	var description, company string
	err := tx.QueryRow(ctx, `SELECT j.title,COALESCE(j.description,''),COALESCE(c.name,'') FROM jobs j LEFT JOIN companies c ON c.id=j.company_id AND c.user_id=j.user_id WHERE j.id=$1 AND j.user_id=$2`, job, owner).Scan(&result.JobTitle, &description, &company)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	if !validText(description, 30000, true) {
		return result, ErrValidation
	}
	err = tx.QueryRow(ctx, `SELECT title FROM resume_builders WHERE id=$1 AND user_id=$2`, resume, owner).Scan(&result.ResumeTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	result.Job = result.JobTitle + "\n" + company + "\n" + description
	hidden := map[string]bool{}
	rows, err := tx.Query(ctx, `SELECT section_key FROM resume_section_orders WHERE resume_builder_id=$1 AND is_visible=false`, resume)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return result, err
		}
		hidden[key] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, err
	}
	for _, section := range sourceSections {
		if (section.key == "custom_sections" && hidden["custom"]) || hidden[section.key] || (section.key == "experience" && hidden["experiences"]) || (section.key == "education" && hidden["educations"]) {
			continue
		}
		// security: identifiers are exclusively from the static sourceSections list.
		columns := strings.Split(section.columns, ",")
		for i, col := range columns {
			label := strings.TrimSuffix(col, "::text")
			columns[i] = "'" + label + ": ' || " + col
		}
		labelled := strings.Join(columns, ",")
		rows, err := tx.Query(ctx, `SELECT id::text,concat_ws(E'\n',`+labelled+`) FROM `+section.table+` WHERE resume_builder_id=$1 ORDER BY id LIMIT 201`, resume)
		if err != nil {
			return result, err
		}
		count := 0
		for rows.Next() {
			var id, text string
			if err := rows.Scan(&id, &text); err != nil {
				rows.Close()
				return result, err
			}
			count++
			if count > 200 || !validText(text, 12000, false) {
				rows.Close()
				return result, ErrValidation
			}
			if strings.TrimSpace(text) != "" {
				result.Facts = append(result.Facts, Fact{ID: section.key + ":" + id, Section: section.key, Text: text})
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return result, err
		}
	}
	data, _ := json.Marshal(result.Facts)
	if len(data) > 60000 {
		return result, ErrValidation
	}
	result.Hash = digest(struct {
		Job, Title string
		Facts      []Fact
	}{result.Job, result.ResumeTitle, result.Facts})
	return result, nil
}

// Source parent locks prevent FK inserts while row share locks protect existing
// content/visibility. This makes the final hash check and publication atomic.
func lockSource(ctx context.Context, tx pgx.Tx, owner, job, resume string) error {
	for _, entry := range []struct{ table, id string }{{"jobs", job}, {"resume_builders", resume}} {
		var id string
		if err := tx.QueryRow(ctx, `SELECT id FROM `+entry.table+` WHERE id=$1 AND user_id=$2 FOR UPDATE`, entry.id, owner).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
	}
	tables := []string{"resume_section_orders"}
	for _, section := range sourceSections {
		tables = append(tables, section.table)
	}
	for _, table := range tables {
		rows, err := tx.Query(ctx, `SELECT id FROM `+table+` WHERE resume_builder_id=$1 ORDER BY id FOR SHARE`, resume)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	// Company name is material source context too.
	rows, err := tx.Query(ctx, `SELECT c.id FROM companies c JOIN jobs j ON j.company_id=c.id AND j.user_id=c.user_id WHERE j.id=$1 AND j.user_id=$2 FOR SHARE OF c`, job, owner)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
	}
	return rows.Err()
}
