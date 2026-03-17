//go:build integration

package e2e

import (
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/andreypavlenko/jobber/internal/platform/postgres"

	authHandler "github.com/andreypavlenko/jobber/modules/auth/handler"
	authRepo "github.com/andreypavlenko/jobber/modules/auth/repository"
	authService "github.com/andreypavlenko/jobber/modules/auth/service"
	userRepo "github.com/andreypavlenko/jobber/modules/users/repository"

	appHandler "github.com/andreypavlenko/jobber/modules/applications/handler"
	appRepo "github.com/andreypavlenko/jobber/modules/applications/repository"
	appService "github.com/andreypavlenko/jobber/modules/applications/service"

	companyHandler "github.com/andreypavlenko/jobber/modules/companies/handler"
	companyRepo "github.com/andreypavlenko/jobber/modules/companies/repository"
	companyService "github.com/andreypavlenko/jobber/modules/companies/service"

	jobHandler "github.com/andreypavlenko/jobber/modules/jobs/handler"
	jobRepo "github.com/andreypavlenko/jobber/modules/jobs/repository"
	jobService "github.com/andreypavlenko/jobber/modules/jobs/service"

	resumeHandler "github.com/andreypavlenko/jobber/modules/resumes/handler"
	resumeRepo "github.com/andreypavlenko/jobber/modules/resumes/repository"
	resumeService "github.com/andreypavlenko/jobber/modules/resumes/service"

	commentHandler "github.com/andreypavlenko/jobber/modules/comments/handler"
	commentRepo "github.com/andreypavlenko/jobber/modules/comments/repository"
	commentService "github.com/andreypavlenko/jobber/modules/comments/service"

	analyticsHandler "github.com/andreypavlenko/jobber/modules/analytics/handler"
	analyticsRepo "github.com/andreypavlenko/jobber/modules/analytics/repository"
	analyticsService "github.com/andreypavlenko/jobber/modules/analytics/service"

	matchScoreRepo "github.com/andreypavlenko/jobber/modules/matchscore/repository"

	clHandler "github.com/andreypavlenko/jobber/modules/contentlibrary/handler"
	clRepo "github.com/andreypavlenko/jobber/modules/contentlibrary/repository"
	clService "github.com/andreypavlenko/jobber/modules/contentlibrary/service"

	cvHandler "github.com/andreypavlenko/jobber/modules/coverletters/handler"
	cvRepo "github.com/andreypavlenko/jobber/modules/coverletters/repository"
	cvService "github.com/andreypavlenko/jobber/modules/coverletters/service"

	rbHandler "github.com/andreypavlenko/jobber/modules/resumebuilder/handler"
	rbRepo "github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	rbService "github.com/andreypavlenko/jobber/modules/resumebuilder/service"

	subHandler "github.com/andreypavlenko/jobber/modules/subscriptions/handler"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	goRedis "github.com/redis/go-redis/v9"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Package-level vars shared across all test files.
var (
	serverURL  string
	pool       *pgxpool.Pool
	rdb        *goRedis.Client
	jwtManager *auth.JWTManager
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// --- Start PostgreSQL container ---
	pgContainer, err := tcPostgres.Run(ctx,
		"postgres:16-alpine",
		tcPostgres.WithDatabase("jobber_test"),
		tcPostgres.WithUsername("test"),
		tcPostgres.WithPassword("test"),
		tcPostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	pgHost, _ := pgContainer.Host(ctx)
	pgPort, _ := pgContainer.MappedPort(ctx, "5432")

	dbCfg := config.DatabaseConfig{
		Host:            pgHost,
		Port:            pgPort.Port(),
		User:            "test",
		Password:        "test",
		DBName:          "jobber_test",
		SSLMode:         "disable",
		MaxConns:        10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// --- Start Redis container ---
	redisContainer, err := tcRedis.Run(ctx,
		"redis:7-alpine",
	)
	if err != nil {
		log.Fatalf("failed to start redis container: %v", err)
	}
	defer redisContainer.Terminate(ctx) //nolint:errcheck

	redisHost, _ := redisContainer.Host(ctx)
	redisPort, _ := redisContainer.MappedPort(ctx, "6379")

	// --- Run migrations ---
	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		log.Fatalf("failed to resolve migrations path: %v", err)
	}

	zapLogger, err := logger.New("warn", "console")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	if err := postgres.RunMigrations(ctx, dbCfg, zapLogger, migrationsPath); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// --- Create pgxpool ---
	poolCfg, err := pgxpool.ParseConfig(dbCfg.DSN())
	if err != nil {
		log.Fatalf("failed to parse pool config: %v", err)
	}
	poolCfg.MaxConns = 10
	pool, err = pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	// --- Create Redis client ---
	rdb = goRedis.NewClient(&goRedis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})
	defer rdb.Close()

	// --- Build router (mirrors cmd/api/main.go) ---
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpPlatform.RequestIDMiddleware())

	// Health + ping (inline, same as main.go)
	pgClient := &postgres.Client{Pool: pool}
	router.GET("/health", func(c *gin.Context) {
		services := make(map[string]string)
		if err := pgClient.Health(ctx); err != nil {
			services["postgres"] = "down"
		} else {
			services["postgres"] = "up"
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			services["redis"] = "down"
		} else {
			services["redis"] = "up"
		}
		httpPlatform.RespondWithHealth(c, services)
	})
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// JWT manager
	jwtManager = auth.NewJWTManager(
		"test-access-secret-minimum-32chars!!",
		"test-refresh-secret-minimum-32chars!",
		15*time.Minute,
		7*24*time.Hour,
	)
	authMiddleware := auth.AuthMiddleware(jwtManager)

	// Rate limiter with high limits
	authRateLimiter := httpPlatform.RateLimitMiddleware(rdb, httpPlatform.RateLimitConfig{
		MaxRequests: 1000,
		Window:      1 * time.Minute,
		KeyPrefix:   "test_auth",
	}, zapLogger.Logger)

	// Repositories
	userRepository := userRepo.NewUserRepository(pool)
	tokenRepository := authRepo.NewRefreshTokenRepository(pool)
	companyRepository := companyRepo.NewCompanyRepository(pool)
	jobRepository := jobRepo.NewJobRepository(pool)
	resumeRepository := resumeRepo.NewResumeRepository(pool)
	applicationRepository := appRepo.NewApplicationRepository(pool)
	stageTemplateRepository := appRepo.NewStageTemplateRepository(pool)
	applicationStageRepository := appRepo.NewApplicationStageRepository(pool)
	commentRepository := commentRepo.NewCommentRepository(pool)
	analyticsRepository := analyticsRepo.NewAnalyticsRepository(pool)
	subscriptionRepository := subRepo.NewSubscriptionRepository(pool)
	matchScoreCacheRepository := matchScoreRepo.NewMatchScoreCacheRepository(pool)

	// Services
	subscriptionSvc := subService.NewSubscriptionService(
		subscriptionRepository,
		"", "", "", "", "", "sandbox",
	)

	authSvc := authService.NewAuthService(
		userRepository,
		tokenRepository,
		jwtManager,
		15*time.Minute,
		7*24*time.Hour,
		subscriptionSvc,
	)
	companySvc := companyService.NewCompanyService(companyRepository)
	jobSvc := jobService.NewJobService(jobRepository, companyRepository, subscriptionSvc, matchScoreCacheRepository)
	resumeSvc := resumeService.NewResumeService(resumeRepository, nil, subscriptionSvc, matchScoreCacheRepository)
	applicationSvc := appService.NewApplicationService(
		pool,
		applicationRepository,
		applicationStageRepository,
		stageTemplateRepository,
		jobRepository,
		companyRepository,
		resumeRepository,
		commentRepository,
		zapLogger,
		subscriptionSvc,
	)
	commentSvc := commentService.NewCommentService(commentRepository)
	analyticsSvc := analyticsService.NewAnalyticsService(analyticsRepository)

	resumeBuilderRepository := rbRepo.NewResumeBuilderRepository(pool)
	resumeBuilderSvc := rbService.NewResumeBuilderService(resumeBuilderRepository, subscriptionSvc)

	contentLibraryRepository := clRepo.NewContentLibraryRepository(pool)
	contentLibrarySvc := clService.NewContentLibraryService(contentLibraryRepository)

	coverLetterRepository := cvRepo.NewCoverLetterRepository(pool)
	coverLetterSvc := cvService.NewCoverLetterService(coverLetterRepository, subscriptionSvc)

	// Handlers
	authHdl := authHandler.NewAuthHandler(authSvc)
	companyHdl := companyHandler.NewCompanyHandler(companySvc)
	jobHdl := jobHandler.NewJobHandler(jobSvc)
	resumeHdl := resumeHandler.NewResumeHandler(resumeSvc)
	applicationHdl := appHandler.NewApplicationHandler(applicationSvc)
	commentHdl := commentHandler.NewCommentHandler(commentSvc)
	analyticsHdl := analyticsHandler.NewAnalyticsHandler(analyticsSvc)
	resumeBuilderHdl := rbHandler.NewResumeBuilderHandler(resumeBuilderSvc)
	contentLibraryHdl := clHandler.NewContentLibraryHandler(contentLibrarySvc)
	coverLetterHdl := cvHandler.NewCoverLetterHandler(coverLetterSvc)
	subscriptionHdl := subHandler.NewSubscriptionHandler(subscriptionSvc, zapLogger.Logger)
	webhookHdl := subHandler.NewWebhookHandler(subscriptionSvc)

	// Register routes
	v1 := router.Group("/api/v1")
	{
		authHdl.RegisterRoutes(v1, authHandler.AuthRouteConfig{
			AuthMiddleware:   authMiddleware,
			RateLimiter:      authRateLimiter,
		})
		companyHdl.RegisterRoutes(v1, authMiddleware)
		jobHdl.RegisterRoutes(v1, authMiddleware)
		resumeHdl.RegisterRoutes(v1, authMiddleware)
		applicationHdl.RegisterRoutes(v1, authMiddleware)
		commentHdl.RegisterRoutes(v1, authMiddleware)
		analyticsHdl.RegisterRoutes(v1, authMiddleware)
		resumeBuilderHdl.RegisterRoutes(v1, authMiddleware)
		contentLibraryHdl.RegisterRoutes(v1, authMiddleware)
		coverLetterHdl.RegisterRoutes(v1, authMiddleware)
		subscriptionHdl.RegisterRoutes(v1, authMiddleware)
		webhookHdl.RegisterRoutes(v1)
	}

	// Start test server
	ts := httptest.NewServer(router)
	serverURL = ts.URL
	defer ts.Close()

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// waitStrategy returns a wait strategy for postgres that waits for the port to be ready.
func waitStrategy() *wait.HostPortStrategy {
	return wait.ForListeningPort("5432/tcp")
}
