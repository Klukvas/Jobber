// Jobber API server entrypoint
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/andreypavlenko/jobber/docs" // swagger docs

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/docx"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/andreypavlenko/jobber/internal/platform/postgres"
	"github.com/andreypavlenko/jobber/internal/platform/redis"
	"github.com/andreypavlenko/jobber/internal/platform/email"
	sentryPlatform "github.com/andreypavlenko/jobber/internal/platform/sentry"
	"github.com/andreypavlenko/jobber/internal/platform/storage"

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

	calendarHandler "github.com/andreypavlenko/jobber/modules/calendar/handler"
	calendarRepo "github.com/andreypavlenko/jobber/modules/calendar/repository"
	calendarService "github.com/andreypavlenko/jobber/modules/calendar/service"

	importHandler "github.com/andreypavlenko/jobber/modules/jobimport/handler"
	importService "github.com/andreypavlenko/jobber/modules/jobimport/service"

	matchScoreHandler "github.com/andreypavlenko/jobber/modules/matchscore/handler"
	matchScoreRepo "github.com/andreypavlenko/jobber/modules/matchscore/repository"
	matchScoreService "github.com/andreypavlenko/jobber/modules/matchscore/service"

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
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// @title Jobber API
// @version 1.0
// @description Job Application Tracking Platform API - A modular monolith backend for managing job applications, companies, resumes, and application stages.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@jobber.example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @x-extension-openapi {"example": "value on a json format"}

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Apply plan limits from YAML config (if loaded)
	if cfg.Plans != nil {
		plans := make(map[string]subModel.PlanLimitsConfig, len(cfg.Plans))
		for k, v := range cfg.Plans {
			plans[k] = subModel.PlanLimitsConfig{
				MaxJobs:           v.MaxJobs,
				MaxResumes:        v.MaxResumes,
				MaxApplications:   v.MaxApplications,
				MaxAIRequests:     v.MaxAIRequests,
				MaxJobParses:      v.MaxJobParses,
				MaxResumeBuilders: v.MaxResumeBuilders,
				MaxCoverLetters:   v.MaxCoverLetters,
			}
		}
		subModel.ApplyPlansConfig(plans)
		log.Printf("Plan limits loaded from YAML config")
	}

	// Initialize logger
	logger, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize Sentry (respects feature flag)
	var sentryEnabled bool
	if cfg.Features.SentryEnabled {
		sentryEnabled = sentryPlatform.Init(cfg.Sentry.DSN, cfg.Server.Env, cfg.Sentry.Release, logger.Logger)
		if sentryEnabled {
			defer sentryPlatform.Flush()
		}
	} else {
		logger.Info("Sentry disabled via FEATURE_SENTRY_ENABLED=false")
	}

	logger.Info("Starting Jobber API server",
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	ctx := context.Background()

	// Initialize PostgreSQL
	pgClient, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer pgClient.Close()
	logger.Info("Connected to PostgreSQL")

	// Run database migrations (MANDATORY: must run before HTTP server starts)
	migrationsPath := "./migrations"
	if err := postgres.RunMigrations(ctx, cfg.Database, logger, migrationsPath); err != nil {
		logger.Fatal("Failed to run database migrations",
			zap.Error(err),
			zap.String("migrations_path", migrationsPath),
		)
	}

	// Initialize Redis
	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// Initialize S3 client (optional - gracefully handle missing config)
	var s3Client *storage.S3Client
	if cfg.S3.Endpoint != "" && cfg.S3.Bucket != "" {
		s3Client, err = storage.NewS3Client(cfg.S3)
		if err != nil {
			logger.Warn("Failed to initialize S3 client, file upload will be disabled", zap.Error(err))
		} else {
			logger.Info("S3 client initialized", zap.String("bucket", cfg.S3.Bucket))
		}
	} else {
		logger.Info("S3 configuration not provided, file upload will be disabled")
	}

	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.New()
	router.Use(sentryPlatform.RecoveryMiddleware(sentryEnabled))
	router.Use(httpPlatform.RequestIDMiddleware())
	router.Use(httpPlatform.LoggerMiddleware(logger))
	router.Use(httpPlatform.CORSMiddleware(cfg.Server.AllowedOrigins))

	// Swagger documentation (available in development)
	if cfg.Server.Env != "production" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		logger.Info("Swagger UI available at /swagger/index.html")
	}

	// Health check endpoint
	router.GET("/health", healthCheckHandler(ctx, pgClient, redisClient))
	
	// Ping endpoint
	router.GET("/ping", pingHandler)

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	// Auth middleware
	authMiddleware := auth.AuthMiddleware(jwtManager)

	// Initialize email sender (respects feature flag)
	var emailSender email.Sender
	if !cfg.Features.EmailEnabled {
		emailSender = &email.NoopSender{Logger: logger.Logger}
		logger.Info("Email disabled via FEATURE_EMAIL_ENABLED=false, using no-op sender")
	} else if cfg.Resend.APIKey != "" {
		emailSender = email.NewResendSender(cfg.Resend.APIKey, cfg.Resend.FromAddress)
		logger.Info("Resend email sender initialized")
	} else {
		emailSender = &email.NoopSender{Logger: logger.Logger}
		logger.Info("Resend not configured, using no-op email sender")
	}

	// Initialize repositories
	userRepository := userRepo.NewUserRepository(pgClient.Pool)
	tokenRepository := authRepo.NewRefreshTokenRepository(pgClient.Pool)
	companyRepository := companyRepo.NewCompanyRepository(pgClient.Pool)
	jobRepository := jobRepo.NewJobRepository(pgClient.Pool)
	resumeRepository := resumeRepo.NewResumeRepository(pgClient.Pool)
	applicationRepository := appRepo.NewApplicationRepository(pgClient.Pool)
	stageTemplateRepository := appRepo.NewStageTemplateRepository(pgClient.Pool)
	applicationStageRepository := appRepo.NewApplicationStageRepository(pgClient.Pool)
	commentRepository := commentRepo.NewCommentRepository(pgClient.Pool)
	analyticsRepository := analyticsRepo.NewAnalyticsRepository(pgClient.Pool)
	subscriptionRepository := subRepo.NewSubscriptionRepository(pgClient.Pool)

	// Initialize subscription service (used as limit checker by other services)
	subscriptionSvc := subService.NewSubscriptionService(
		subscriptionRepository,
		cfg.Paddle.WebhookSecret,
		cfg.Paddle.APIKey,
		cfg.Paddle.ProPriceID,
		cfg.Paddle.EnterprisePriceID,
		cfg.Paddle.ClientToken,
		cfg.Paddle.Environment,
	)

	// Initialize match score cache repository
	matchScoreCacheRepo := matchScoreRepo.NewMatchScoreCacheRepository(pgClient.Pool)

	// Initialize verification and password reset repositories
	verificationRepository := authRepo.NewEmailVerificationRepository(pgClient.Pool)
	passwordResetRepository := authRepo.NewPasswordResetRepository(pgClient.Pool)

	// Initialize services
	authSvc := authService.NewAuthService(authService.AuthServiceConfig{
		UserRepo:            userRepository,
		TokenRepo:           tokenRepository,
		VerificationRepo:    verificationRepository,
		PasswordResetRepo:   passwordResetRepository,
		EmailSender:         emailSender,
		JWTManager:          jwtManager,
		AccessExpiry:        cfg.JWT.AccessExpiry,
		RefreshExpiry:       cfg.JWT.RefreshExpiry,
		SubscriptionCreator: subscriptionSvc,
		Logger:              logger.Logger,
	})
	companySvc := companyService.NewCompanyService(companyRepository)
	jobSvc := jobService.NewJobService(jobRepository, companyRepository, subscriptionSvc, matchScoreCacheRepo)
	resumeSvc := resumeService.NewResumeService(resumeRepository, s3Client, subscriptionSvc, matchScoreCacheRepo)

	// Initialize resume builder repository early — needed by application service
	resumeBuilderRepository := rbRepo.NewResumeBuilderRepository(pgClient.Pool)

	applicationSvc := appService.NewApplicationService(
		pgClient.Pool,
		applicationRepository,
		applicationStageRepository,
		stageTemplateRepository,
		jobRepository,
		companyRepository,
		resumeRepository,
		resumeBuilderRepository,
		commentRepository,
		logger,
		subscriptionSvc,
	)
	commentSvc := commentService.NewCommentService(commentRepository)
	analyticsSvc := analyticsService.NewAnalyticsService(analyticsRepository)

	// Initialize handlers
	cookieCfg := auth.NewCookieConfig(cfg.Server.Env)
	authHdl := authHandler.NewAuthHandler(authSvc, cookieCfg, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)
	companyHdl := companyHandler.NewCompanyHandler(companySvc)
	jobHdl := jobHandler.NewJobHandler(jobSvc)
	resumeHdl := resumeHandler.NewResumeHandler(resumeSvc)
	applicationHdl := appHandler.NewApplicationHandler(applicationSvc)
	commentHdl := commentHandler.NewCommentHandler(commentSvc)
	analyticsHdl := analyticsHandler.NewAnalyticsHandler(analyticsSvc)
	subscriptionHdl := subHandler.NewSubscriptionHandler(subscriptionSvc, logger.Logger)
	webhookHdl := subHandler.NewWebhookHandler(subscriptionSvc, logger.Logger)

	// Initialize calendar module (optional — only if all Google Calendar config is provided)
	var calendarHdl *calendarHandler.CalendarHandler
	if cfg.GoogleCalendar.ClientID != "" &&
		cfg.GoogleCalendar.ClientSecret != "" &&
		cfg.GoogleCalendar.TokenEncryptionKey != "" &&
		cfg.GoogleCalendar.RedirectURL != "" &&
		cfg.GoogleCalendar.FrontendURL != "" {
		oauthConfig := &oauth2.Config{
			ClientID:     cfg.GoogleCalendar.ClientID,
			ClientSecret: cfg.GoogleCalendar.ClientSecret,
			RedirectURL:  cfg.GoogleCalendar.RedirectURL,
			Scopes:       []string{calendar.CalendarEventsScope},
			Endpoint:     google.Endpoint,
		}

		encryptor, err := calendarService.NewEncryptor(cfg.GoogleCalendar.TokenEncryptionKey)
		if err != nil {
			logger.Fatal("Failed to initialize calendar encryption", zap.Error(err))
		}

		calTokenRepo := calendarRepo.NewTokenRepository(pgClient.Pool)
		calStageRepo := calendarRepo.NewStageRepository(pgClient.Pool)
		gcalClient := calendarService.NewGoogleClient(oauthConfig)
		calSvc := calendarService.NewCalendarService(
			calTokenRepo,
			calStageRepo,
			gcalClient,
			encryptor,
			oauthConfig,
			redisClient.Client,
			cfg.GoogleCalendar.FrontendURL,
		)
		calendarHdl = calendarHandler.NewCalendarHandler(calSvc)
		logger.Info("Google Calendar integration enabled")
	} else {
		logger.Info("Google Calendar not configured, integration disabled")
	}

	// Initialize resume builder module
	resumeBuilderSvc := rbService.NewResumeBuilderService(resumeBuilderRepository, subscriptionSvc)
	resumeBuilderHdl := rbHandler.NewResumeBuilderHandler(resumeBuilderSvc)

	// Initialize content library module
	contentLibraryRepository := clRepo.NewContentLibraryRepository(pgClient.Pool)
	contentLibrarySvc := clService.NewContentLibraryService(contentLibraryRepository)
	contentLibraryHdl := clHandler.NewContentLibraryHandler(contentLibrarySvc)

	// Initialize cover letter module
	coverLetterRepository := cvRepo.NewCoverLetterRepository(pgClient.Pool)
	coverLetterSvc := cvService.NewCoverLetterService(coverLetterRepository, subscriptionSvc)
	coverLetterHdl := cvHandler.NewCoverLetterHandler(coverLetterSvc)

	// Initialize PDF service for resume export (optional — requires headless Chrome)
	var exportHdl *rbHandler.ExportHandler
	var coverLetterExportHdl *cvHandler.ExportHandler
	pdfSvc, pdfErr := pdf.NewPDFService(logger.Logger, cfg.Server.FrontendURL)
	if pdfErr != nil {
		logger.Warn("PDF service not available, export disabled", zap.Error(pdfErr))
	} else {
		defer pdfSvc.Close()
		exportHdl = rbHandler.NewExportHandler(resumeBuilderSvc, pdfSvc, logger.Logger)
		coverLetterExportHdl = cvHandler.NewExportHandler(coverLetterSvc, pdfSvc)
		if pdfSvc.HasFrontendPDF() {
			logger.Info("PDF service initialized with React frontend rendering",
				zap.String("frontend_url", cfg.Server.FrontendURL),
			)
		} else {
			logger.Info("PDF service initialized with Go templates only (set FRONTEND_URL to enable React rendering)")
		}
	}

	// Initialize DOCX service for resume and cover letter export
	docxSvc := docx.NewDOCXService()
	if exportHdl != nil {
		exportHdl.SetDOCXService(docxSvc)
	}
	if coverLetterExportHdl != nil {
		coverLetterExportHdl.SetDOCXService(docxSvc)
	}

	// Initialize AI client for job import, match scoring, and resume AI (optional — only if ANTHROPIC_API_KEY is set)
	var importHdl *importHandler.ImportHandler
	var matchScoreHdl *matchScoreHandler.MatchScoreHandler
	var resumeAIHdl *rbHandler.AIHandler
	var resumeImportHdl *rbHandler.ImportHandler
	var coverLetterAIHdl *cvHandler.AIHandler
	if cfg.Anthropic.APIKey != "" {
		aiClient := ai.NewAnthropicClient(cfg.Anthropic.APIKey)
		importSvc := importService.NewImportService(aiClient, subscriptionSvc)
		importHdl = importHandler.NewImportHandler(importSvc)

		matchScoreSvc := matchScoreService.NewMatchScoreService(aiClient, s3Client, jobRepository, resumeRepository, subscriptionSvc, matchScoreCacheRepo)
		matchScoreHdl = matchScoreHandler.NewMatchScoreHandler(matchScoreSvc)

		resumeAISvc := rbService.NewAIService(resumeBuilderRepository, aiClient, subscriptionSvc)
		resumeAIHdl = rbHandler.NewAIHandler(resumeAISvc)

		resumeImportSvc := rbService.NewImportService(resumeBuilderRepository, aiClient, subscriptionSvc)
		resumeImportHdl = rbHandler.NewImportHandler(resumeImportSvc)

		coverLetterAISvc := cvService.NewAIService(coverLetterRepository, resumeBuilderRepository, aiClient, subscriptionSvc)
		coverLetterAIHdl = cvHandler.NewAIHandler(coverLetterAISvc, logger.Logger)

		logger.Info("AI job import, match scoring, resume AI, and cover letter AI enabled")
	} else {
		logger.Info("ANTHROPIC_API_KEY not configured, AI features disabled")
	}

	// Rate limiting for auth endpoints (10 requests per minute per IP)
	authRateLimiter := httpPlatform.RateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 10,
		Window:      1 * time.Minute,
		KeyPrefix:   "auth",
	}, logger.Logger)

	// Per-user rate limiting for authenticated AI/export endpoints
	importRateLimiter := httpPlatform.UserRateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 20,
		Window:      1 * time.Minute,
		KeyPrefix:   "ai_import",
	}, logger.Logger)

	matchScoreRateLimiter := httpPlatform.UserRateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 10,
		Window:      1 * time.Minute,
		KeyPrefix:   "match_score",
	}, logger.Logger)

	exportRateLimiter := httpPlatform.UserRateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 5,
		Window:      1 * time.Minute,
		KeyPrefix:   "pdf_export",
	}, logger.Logger)

	resumeAIRateLimiter := httpPlatform.UserRateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 20,
		Window:      1 * time.Minute,
		KeyPrefix:   "resume_ai",
	}, logger.Logger)

	resumeImportRateLimiter := httpPlatform.UserRateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 20,
		Window:      1 * time.Minute,
		KeyPrefix:   "resume_import",
	}, logger.Logger)

	coverLetterAIRateLimiter := httpPlatform.UserRateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 20,
		Window:      1 * time.Minute,
		KeyPrefix:   "cover_letter_ai",
	}, logger.Logger)

	// Stricter rate limiting for email-sending endpoints (3 requests per 15 minutes per IP)
	emailRateLimiter := httpPlatform.RateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 3,
		Window:      15 * time.Minute,
		KeyPrefix:   "email_send",
	}, logger.Logger)

	// Stricter rate limiting for code verification endpoints (5 requests per 5 minutes per IP)
	codeRateLimiter := httpPlatform.RateLimitMiddleware(redisClient.Client, httpPlatform.RateLimitConfig{
		MaxRequests: 5,
		Window:      5 * time.Minute,
		KeyPrefix:   "code_verify",
	}, logger.Logger)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Register module routes
		authHdl.RegisterRoutes(v1, authHandler.AuthRouteConfig{
			AuthMiddleware:   authMiddleware,
			RateLimiter:      authRateLimiter,
			EmailRateLimiter: emailRateLimiter,
			CodeRateLimiter:  codeRateLimiter,
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
		subscriptionHdl.RegisterRoutes(v1, authMiddleware, cfg.Features.PaymentsEnabled)
		if cfg.Features.PaymentsEnabled {
			webhookHdl.RegisterRoutes(v1) // Public, no auth — Paddle calls this
		} else {
			logger.Info("Payments disabled via FEATURE_PAYMENTS_ENABLED=false, Paddle webhook and checkout routes not registered")
		}
		if calendarHdl != nil {
			calendarHdl.RegisterRoutes(v1, authMiddleware)
		}
		if importHdl != nil {
			importHdl.RegisterRoutes(v1, authMiddleware, importRateLimiter)
		}
		if matchScoreHdl != nil {
			matchScoreHdl.RegisterRoutes(v1, authMiddleware, matchScoreRateLimiter)
		}
		if exportHdl != nil {
			exportHdl.RegisterRoutes(v1, authMiddleware, exportRateLimiter)
		}
		if coverLetterExportHdl != nil {
			coverLetterExportHdl.RegisterRoutes(v1, authMiddleware, exportRateLimiter)
		}
		if resumeAIHdl != nil {
			resumeAIHdl.RegisterRoutes(v1, authMiddleware, resumeAIRateLimiter)
		}
		if resumeImportHdl != nil {
			resumeImportHdl.RegisterRoutes(v1, authMiddleware, resumeImportRateLimiter)
		}
		if coverLetterAIHdl != nil {
			coverLetterAIHdl.RegisterRoutes(v1, authMiddleware, coverLetterAIRateLimiter)
		}
	}

	// Start background job: clean up expired tokens every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			bgCtx := context.Background()
			if err := tokenRepository.DeleteExpired(bgCtx); err != nil {
				logger.Error("Failed to clean up expired refresh tokens", zap.Error(err))
			}
			if err := verificationRepository.DeleteExpired(bgCtx); err != nil {
				logger.Error("Failed to clean up expired verification tokens", zap.Error(err))
			}
			if err := passwordResetRepository.DeleteExpired(bgCtx); err != nil {
				logger.Error("Failed to clean up expired password reset tokens", zap.Error(err))
			}
			logger.Debug("Expired tokens cleaned up")
		}
	}()

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server listening", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// healthCheckHandler godoc
// @Summary Health Check
// @Description Check the health status of the application and its dependencies
// @Tags system
// @Produce json
// @Success 200 {object} http.HealthResponse
// @Router /health [get]
func healthCheckHandler(ctx context.Context, pgClient *postgres.Client, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		services := make(map[string]string)

		// Check PostgreSQL
		if err := pgClient.Health(ctx); err != nil {
			services["postgres"] = "down"
		} else {
			services["postgres"] = "up"
		}

		// Check Redis
		if err := redisClient.Health(ctx); err != nil {
			services["redis"] = "down"
		} else {
			services["redis"] = "up"
		}

		httpPlatform.RespondWithHealth(c, services)
	}
}

// pingHandler godoc
// @Summary Ping
// @Description Simple ping endpoint to check if the API is responding
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Router /ping [get]
func pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
