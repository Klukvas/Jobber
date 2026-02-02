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
	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/andreypavlenko/jobber/internal/platform/postgres"
	"github.com/andreypavlenko/jobber/internal/platform/redis"
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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
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

	// Initialize logger
	logger, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

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
	router.Use(gin.Recovery())
	router.Use(httpPlatform.RequestIDMiddleware())
	router.Use(httpPlatform.LoggerMiddleware(logger))
	router.Use(httpPlatform.CORSMiddleware())

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

	// Initialize services
	authSvc := authService.NewAuthService(
		userRepository,
		tokenRepository,
		jwtManager,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)
	companySvc := companyService.NewCompanyService(companyRepository)
	jobSvc := jobService.NewJobService(jobRepository)
	resumeSvc := resumeService.NewResumeService(resumeRepository, s3Client)
	applicationSvc := appService.NewApplicationService(
		applicationRepository,
		applicationStageRepository,
		stageTemplateRepository,
		jobRepository,
		companyRepository,
		resumeRepository,
		commentRepository,
	)
	commentSvc := commentService.NewCommentService(commentRepository)

	// Initialize handlers
	authHdl := authHandler.NewAuthHandler(authSvc)
	companyHdl := companyHandler.NewCompanyHandler(companySvc)
	jobHdl := jobHandler.NewJobHandler(jobSvc)
	resumeHdl := resumeHandler.NewResumeHandler(resumeSvc)
	applicationHdl := appHandler.NewApplicationHandler(applicationSvc)
	commentHdl := commentHandler.NewCommentHandler(commentSvc)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Register module routes
		authHdl.RegisterRoutes(v1)
		companyHdl.RegisterRoutes(v1, authMiddleware)
		jobHdl.RegisterRoutes(v1, authMiddleware)
		resumeHdl.RegisterRoutes(v1, authMiddleware)
		applicationHdl.RegisterRoutes(v1, authMiddleware)
		commentHdl.RegisterRoutes(v1, authMiddleware)
	}

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
