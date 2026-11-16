//go:build integration

package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
	matchRepo "github.com/andreypavlenko/jobber/modules/matchscore/repository"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subSvc "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newImportCreditDB(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("R3_CREDIT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("R3_CREDIT_TEST_DATABASE_URL must point to jobber_r3_credit_test")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "jobber_r3_credit_test", cfg.ConnConfig.Database, "refusing writes outside the dedicated credit test database")
	admin, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	schema := "import_credit_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	_, err = admin.Exec(t.Context(), `CREATE SCHEMA `+quoted)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.Exec(ctx, `DROP SCHEMA `+quoted+` CASCADE`)
		assert.NoError(t, err)
		admin.Close()
	})
	cfg.MaxConns = 1
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	// Keep the fixture wholly inside its schema. This compatibility function
	// supplies migration 1's UUID default without creating a database extension.
	_, err = db.Exec(t.Context(), `CREATE FUNCTION uuid_generate_v4() RETURNS uuid LANGUAGE SQL AS 'SELECT gen_random_uuid()'`)
	require.NoError(t, err)
	migrations, err := filepath.Glob("../../../migrations/*.up.sql")
	require.NoError(t, err)
	for _, path := range migrations {
		body, err := os.ReadFile(path)
		require.NoError(t, err)
		sql := strings.ReplaceAll(string(body), `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`, "")
		_, err = db.Exec(t.Context(), sql)
		require.NoError(t, err, path)
	}
	user := uuid.NewString()
	_, err = db.Exec(t.Context(), `INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Offline credit fixture','test-only')`, user, user+"@import.example.test")
	require.NoError(t, err)
	return db, user
}

func importCreditContext(t *testing.T, db *pgxpool.Pool, user, kind string, admitted, seed bool) context.Context {
	t.Helper()
	key := uuid.NewString()
	ctx := ai.CreditContext(t.Context(), func() string { return user }, key, kind)
	if admitted {
		ai.MarkCreditAdmission(ctx)
	}
	if seed {
		op, _, found := ai.CurrentCreditOperation(ctx)
		require.True(t, found)
		var grant string
		require.NoError(t, db.QueryRow(ctx, `INSERT INTO credit_grants(user_id,kind,credits) VALUES($1,'free_once',30) RETURNING id`, user).Scan(&grant))
		_, err := db.Exec(ctx, `INSERT INTO ai_credit_operations(user_id,operation_key,kind,fingerprint,grant_id,credits,charge_state) VALUES($1,$2,$3,$4,$5,$6,'reserved')`, user, key, kind, op.Fingerprint, grant, op.Credits)
		require.NoError(t, err)
	}
	return ctx
}

func TestImportCreditResultAtomicPostgres(t *testing.T) {
	for _, scenario := range []string{"success", "save failure", "read failure", "charge failure", "commit failure", "not admitted"} {
		t.Run(scenario, func(t *testing.T) {
			db, user := newImportCreditDB(t)
			ctx := importCreditContext(t, db, user, "resume_import", scenario != "not admitted", scenario != "charge failure")
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if scenario == "commit failure" {
				_, err := db.Exec(ctx, `CREATE TABLE commit_guard(id integer PRIMARY KEY);
                ALTER TABLE resume_builders ADD COLUMN commit_guard integer NOT NULL DEFAULT 1;
                ALTER TABLE resume_builders ADD CONSTRAINT force_commit_failure FOREIGN KEY(commit_guard) REFERENCES commit_guard(id) DEFERRABLE INITIALLY DEFERRED`)
				require.NoError(t, err)
			}
			parsed := sampleParsedResume()
			if scenario == "save failure" {
				parsed.FullName = strings.Repeat("x", 256)
			}
			if scenario == "read failure" {
				_, err := db.Exec(ctx, `DROP TABLE resume_languages`)
				require.NoError(t, err)
				parsed.Languages = nil
			}
			svc := NewImportServiceWithDeps(repository.NewResumeBuilderRepository(db), &MockResumeTextParser{ParseResumeTextFunc: func(context.Context, string) (*ai.ParsedResume, error) { return parsed, nil }}, nil, nil)
			got, err := svc.ImportFromText(ctx, user, "offline fixture", "Imported")
			success := scenario == "success" || scenario == "not admitted"
			if success {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, parsed.FullName, got.Contact.FullName)
			} else {
				require.Error(t, err)
				assert.Nil(t, got)
			}
			var saved int
			require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM resume_builders`).Scan(&saved))
			if success {
				assert.Equal(t, 1, saved)
			} else {
				assert.Zero(t, saved, "failed save/read/charge rolls back the entire resume")
			}
			if scenario != "charge failure" {
				var state string
				require.NoError(t, db.QueryRow(ctx, `SELECT charge_state FROM ai_credit_operations`).Scan(&state))
				if scenario == "success" {
					assert.Equal(t, "charged", state)
				} else {
					assert.Equal(t, "reserved", state)
				}
			}
			_, durable, _ := ai.CurrentCreditOperation(ctx)
			assert.Equal(t, scenario == "success", durable)
			assert.Equal(t, success, ai.CreditResultReady(ctx))
			if !success && scenario != "charge failure" {
				subSvc.NewSubscriptionService(subRepo.NewSubscriptionRepository(db), nil, subSvc.BillingConfig{}).FinishAIRequest(ctx)
				var state string
				require.NoError(t, db.QueryRow(ctx, `SELECT charge_state FROM ai_credit_operations`).Scan(&state))
				assert.Equal(t, "released", state, "deferred cleanup must refund a rolled-back result")
				require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM resume_builders`).Scan(&saved))
				assert.Zero(t, saved)
			}
		})
	}
}

func TestMatchCacheCreditResultAtomicPostgres(t *testing.T) {
	for _, scenario := range []string{"success", "save failure", "charge failure", "not admitted"} {
		t.Run(scenario, func(t *testing.T) {
			db, user := newImportCreditDB(t)
			ctx := importCreditContext(t, db, user, "match", scenario != "not admitted", scenario != "charge failure")
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if scenario == "save failure" {
				_, err := db.Exec(ctx, `ALTER TABLE match_score_cache ADD CONSTRAINT test_save_failure CHECK(false)`)
				require.NoError(t, err)
			}
			stage, job, resume := uuid.NewString(), uuid.NewString(), uuid.NewString()
			_, err := db.Exec(ctx, `INSERT INTO stage_templates(id,user_id,name,"order") VALUES($1,$2,'Saved',0)`, stage, user)
			require.NoError(t, err)
			_, err = db.Exec(ctx, `INSERT INTO jobs(id,user_id,title,current_stage_template_id) VALUES($1,$2,'Job',$3)`, job, user, stage)
			require.NoError(t, err)
			_, err = db.Exec(ctx, `INSERT INTO resumes(id,user_id,title,storage_type) VALUES($1,$2,'Resume','external')`, resume, user)
			require.NoError(t, err)
			err = matchRepo.NewMatchScoreCacheRepository(db).Upsert(ctx, user, job, resume, &matchModel.MatchScoreResponse{OverallScore: 80, Summary: "Offline fixture"})
			success := scenario == "success" || scenario == "not admitted"
			if success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			var saved int
			require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM match_score_cache`).Scan(&saved))
			if success {
				assert.Equal(t, 1, saved)
			} else {
				assert.Zero(t, saved)
			}
			if scenario != "charge failure" {
				var state string
				require.NoError(t, db.QueryRow(ctx, `SELECT charge_state FROM ai_credit_operations`).Scan(&state))
				if scenario == "success" {
					assert.Equal(t, "charged", state)
				} else {
					assert.Equal(t, "reserved", state)
				}
			}
			_, durable, _ := ai.CurrentCreditOperation(ctx)
			assert.Equal(t, scenario == "success", durable)
		})
	}
}

func TestImportTransactionReadsOwnWritesPoolOne(t *testing.T) {
	db, user := newImportCreditDB(t)
	ctx := ai.CreditContext(t.Context(), func() string { return user }, uuid.NewString(), "resume_import")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	repo := repository.NewResumeBuilderRepository(db)
	err := repo.RunInTransaction(ctx, func(txRepo ports.ResumeBuilderRepository) error {
		id, err := writeParsedResume(ctx, txRepo, user, sampleParsedResume(), "Inside transaction", "")
		if err != nil {
			return err
		}
		got, err := txRepo.GetFullResume(ctx, user, id)
		if err != nil {
			return err
		}
		require.Equal(t, id, got.ID)
		ai.MarkCreditResult(ctx)
		return assert.AnError
	})
	require.ErrorIs(t, err, assert.AnError)
	assert.False(t, ai.CreditResultReady(ctx))
	var saved int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM resume_builders`).Scan(&saved))
	assert.Zero(t, saved)
}
