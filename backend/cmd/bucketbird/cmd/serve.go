package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bucketbird/backend/internal/api/auth"
	"bucketbird/backend/internal/api/buckets"
	"bucketbird/backend/internal/api/credentials"
	"bucketbird/backend/internal/api/profile"
	"bucketbird/backend/internal/config"
	"bucketbird/backend/internal/logging"
	"bucketbird/backend/internal/middleware"
	"bucketbird/backend/internal/repository"
	"bucketbird/backend/internal/service"
	"bucketbird/backend/pkg/jwt"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the BucketBird API server",
	Long:  `Starts the HTTP API server for BucketBird, handling all API requests.`,
	Run:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) {
	cfg := config.Load()
	logger := logging.NewLogger(cfg.AppName, cfg.Env)
	logger.Info("starting bucketbird API server")

	ctx := context.Background()

	// Initialize database connection pool
	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		logger.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	// Ping database
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("database connection established")

	// Initialize repositories
	repos := repository.NewRepositories(pool)

	// Initialize JWT token manager
	tokenManager := jwt.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)

	// Initialize services
	authService := service.NewAuthService(
		repos.Users,
		repos.Sessions,
		tokenManager,
		cfg.RefreshTokenTTL,
		logger,
	)

	bucketService := service.NewBucketService(
		repos.Buckets,
		repos.Credentials,
		repos.Users,
		cfg.EncryptionKey,
		logger,
	)

	credentialService := service.NewCredentialService(
		repos.Credentials,
		cfg.EncryptionKey,
		logger,
	)

	profileService := service.NewProfileService(repos.Users)

	// Initialize HTTP handlers
	authHandler := auth.NewHandler(authService, logger, cfg.CookieSecure, cfg.EnableDemoLogin)
	bucketHandler := buckets.NewHandler(bucketService, cfg.EncryptionKey, logger)
	credentialHandler := credentials.NewHandler(credentialService, logger)
	profileHandler := profile.NewHandler(profileService, logger)

	// Setup Chi router
	r := chi.NewRouter()

	// Middleware stack
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders)

	// CORS configuration
	allowCredentials := !cfg.HasWildcardOrigin()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))

	// Health check endpoints
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
	r.Get("/health", healthHandler)
	r.Get("/healthz", healthHandler)

	// Public routes (no auth required)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/demo", authHandler.DemoLogin)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Protected routes (auth required)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(authService))
		r.Use(middleware.DemoReadOnly)

		// Auth endpoints (authenticated)
		r.Get("/auth/me", authHandler.Me)

		// Profile routes
		r.Get("/profile", profileHandler.Get)
		r.Put("/profile", profileHandler.Update)
		r.Put("/profile/password", profileHandler.UpdatePassword)

		// Bucket routes
		r.Route("/buckets", func(r chi.Router) {
			r.Get("/", bucketHandler.List)
			r.Post("/", bucketHandler.Create)
			r.Get("/{id}", bucketHandler.Get)
			r.Put("/{id}", bucketHandler.Update)
			r.Delete("/{id}", bucketHandler.Delete)
			r.Post("/{id}/recalculate-size", bucketHandler.RecalculateSize)

			// Object operations
			r.Get("/{id}/objects", bucketHandler.ListObjects)
			r.Get("/{id}/objects/search", bucketHandler.SearchObjects)
			r.Post("/{id}/objects/upload", bucketHandler.UploadObject)
			r.Post("/{id}/objects/import/youtube", bucketHandler.ImportYouTube)
			r.Get("/{id}/objects/download", bucketHandler.DownloadObject)
			r.Post("/{id}/objects/presign", bucketHandler.PresignObject)
			r.Get("/{id}/objects/metadata", bucketHandler.GetObjectMetadata)
			r.Post("/{id}/objects/folders", bucketHandler.CreateFolder)
			r.Post("/{id}/objects/delete", bucketHandler.DeleteObjects)
			r.Post("/{id}/objects/rename", bucketHandler.RenameObject)
			r.Post("/{id}/objects/copy", bucketHandler.CopyObject)
		})

		// Credential routes
		r.Route("/credentials", func(r chi.Router) {
			r.Get("/", credentialHandler.List)
			r.Post("/", credentialHandler.Create)
			r.Get("/{id}", credentialHandler.Get)
			r.Put("/{id}", credentialHandler.Update)
			r.Delete("/{id}", credentialHandler.Delete)
			r.Get("/{id}/buckets", credentialHandler.DiscoverBuckets)
			r.Post("/{id}/test", credentialHandler.Test)
		})
	})

	// HTTP server configuration
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting HTTP server", slog.String("port", cfg.HTTPPort))
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server error", slog.Any("error", err))
		os.Exit(1)

	case sig := <-shutdown:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))

		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", slog.Any("error", err))
			if err := srv.Close(); err != nil {
				logger.Error("failed to close server", slog.Any("error", err))
			}
		}

		logger.Info("server stopped")
	}
}
