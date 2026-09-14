// Command prune-orphan-resumes reclaims uploaded objects that no resume points
// at.
//
// # Why this exists rather than a bucket lifecycle rule
//
// The browser PUTs a resume straight to object storage with a presigned URL and
// then asks the API to finalize it. Almost every failure in between is already
// cleaned up in the request: a file that is not a PDF, one over the size cap, a
// blank title, a plan that turned out to be full — each of those deletes the
// object before answering. What is left is the case nothing server-side can see:
// the PUT succeeded and the finalize call never arrived, because the tab was
// closed or the connection dropped. That object is unreferenced and invisible.
//
// A bucket lifecycle rule cannot solve it. Uploaded and finalized objects share
// one prefix — `users/<user>/resumes/<resume>.pdf` — because the key is the
// resume's permanent home from the moment it is presigned. An age-based expiry
// on that prefix would delete customers' actual resumes.
//
// So the reference set comes from the database, which is the only thing that
// knows which objects are real. An object is deleted only when no `resumes` row
// names it AND it is older than a safety window, so an upload that is being
// finalized right now is never in scope.
//
// # Running it
//
//	cd be
//	go run ./scripts/prune-orphan-resumes                  # dry run, lists only
//	go run ./scripts/prune-orphan-resumes -apply           # actually deletes
//	go run ./scripts/prune-orphan-resumes -older-than 72h  # widen the window
//
// Reads the same S3_* and DB_* variables as the API (see .env.example). It
// never deletes anything without -apply.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// resumeObjectPrefix is where GenerateUploadURL puts every uploaded resume.
// Nothing outside it is ever considered.
const resumeObjectPrefix = "users/"

// defaultSafetyWindow is how recent an object has to be to be left alone.
//
// The presigned URL is good for five minutes and finalization follows the PUT
// immediately, so anything from the last day is either live or being uploaded
// right now. A day is far past both and short enough to be worth running.
const defaultSafetyWindow = 24 * time.Hour

// candidate is one stored object, as far as this tool is concerned.
type candidate struct {
	key      string
	size     int64
	modified time.Time
}

// isOrphan reports whether an object can be deleted.
//
// Both conditions matter, and for different reasons. Being unreferenced is what
// makes an object useless; being older than the cutoff is what makes it safe —
// an upload whose finalize call is in flight has no row yet either, and
// deleting it would fail a purchase-shaped operation for no reason.
func isOrphan(obj candidate, referenced map[string]struct{}, cutoff time.Time) bool {
	if _, live := referenced[obj.key]; live {
		return false
	}
	return obj.modified.Before(cutoff)
}

func main() {
	apply := flag.Bool("apply", false, "delete the orphans instead of listing them")
	olderThan := flag.Duration("older-than", defaultSafetyWindow,
		"leave objects newer than this alone; an upload being finalized right now has no row yet either")
	flag.Parse()

	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	ctx := context.Background()

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatal("S3_BUCKET is not set")
	}
	client, err := newS3Client()
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseDSN())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	referenced, err := referencedKeys(ctx, pool)
	if err != nil {
		log.Fatalf("read resume keys: %v", err)
	}

	objects, err := listResumeObjects(ctx, client, bucket)
	if err != nil {
		log.Fatalf("list objects: %v", err)
	}

	cutoff := time.Now().UTC().Add(-*olderThan)
	var orphans []candidate
	var reclaimed int64
	for _, obj := range objects {
		if isOrphan(obj, referenced, cutoff) {
			orphans = append(orphans, obj)
			reclaimed += obj.size
		}
	}

	fmt.Printf("%d objects under %q, %d referenced by a resume, %d orphaned (%s)\n",
		len(objects), resumeObjectPrefix, len(referenced), len(orphans), humanBytes(reclaimed))

	if !*apply {
		for _, obj := range orphans {
			fmt.Printf("  would delete %s (%s, last modified %s)\n",
				obj.key, humanBytes(obj.size), obj.modified.Format(time.RFC3339))
		}
		fmt.Println("\nDry run. Re-run with -apply to delete these.")
		return
	}

	deleted := 0
	for _, obj := range orphans {
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(obj.key),
		}); err != nil {
			log.Printf("[WARN] could not delete %s: %v", obj.key, err)
			continue
		}
		deleted++
	}
	fmt.Printf("Deleted %d of %d orphaned objects.\n", deleted, len(orphans))
}

// referencedKeys is every storage key a resume row points at — the whole
// reference set, read in one query so a row written mid-listing cannot be
// missed on both sides.
func referencedKeys(ctx context.Context, pool *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx,
		`SELECT storage_key FROM resumes WHERE storage_type = 's3' AND storage_key IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := map[string]struct{}{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys[key] = struct{}{}
	}
	return keys, rows.Err()
}

func listResumeObjects(ctx context.Context, client *s3.Client, bucket string) ([]candidate, error) {
	var objects []candidate
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(resumeObjectPrefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Contents {
			if item.Key == nil || item.LastModified == nil {
				continue
			}
			objects = append(objects, candidate{
				key:      *item.Key,
				size:     aws.ToInt64(item.Size),
				modified: item.LastModified.UTC(),
			})
		}
	}
	return objects, nil
}

func newS3Client() (*s3.Client, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY must all be set")
	}

	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
			if service != s3.ServiceID {
				return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested for %q", service)
			}
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		})

	cfg := aws.Config{
		Region:                      envOr("S3_REGION", "eu-central"),
		Credentials:                 credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		EndpointResolverWithOptions: resolver,
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true }), nil
}

func databaseDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("DB_HOST", "localhost"),
		envOr("DB_PORT", "5432"),
		envOr("DB_USER", "jobber"),
		envOr("DB_PASSWORD", "jobber"),
		envOr("DB_NAME", "jobber"),
		envOr("DB_SSL_MODE", "disable"),
	)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for size := n / unit; size >= unit; size /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
