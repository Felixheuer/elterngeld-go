package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"elterngeld-portal/config"
	_ "elterngeld-portal/docs" // Import generated docs
	"elterngeld-portal/internal/database"
	"elterngeld-portal/internal/server"
	"elterngeld-portal/pkg/logger"

	"go.uber.org/zap"
)

// @title Elterngeld Portal API
// @version 1.0
// @description REST API für das Elterngeldlotsen-Portal
// @termsOfService http://elterngeld-portal.de/terms

// @contact.name API Support
// @contact.url http://elterngeld-portal.de/support
// @contact.email support@elterngeld-portal.de

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

var (
	initDB  = flag.Bool("init-db", false, "Initialize database with migrations and exit")
	migrate = flag.Bool("migrate", false, "Run database migrations and exit")
	seed    = flag.Bool("seed", false, "Seed database with sample data and exit")
)

func main() {
	flag.Parse()

	// Load configuration
	if err := config.Load(); err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Cfg

	// Initialize logger
	if err := logger.Init(cfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Connect to database
	if err := database.Connect(cfg, logger.Logger); err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Handle CLI commands
	if *initDB {
		handleInitDB(cfg)
		return
	}

	if *migrate {
		handleMigrate(cfg)
		return
	}

	if *seed {
		handleSeed(cfg)
		return
	}

	// Normal server startup
	startServer(cfg)
}

func handleInitDB(cfg *config.Config) {
	logger.Info("Initializing database...")

	// Run auto-migration
	if err := database.AutoMigrate(); err != nil {
		logger.Fatal("Database initialization failed", zap.Error(err))
	}

	logger.Info("Database initialized successfully")

	// If seed data is enabled, also seed
	if cfg.Dev.SeedData {
		if err := database.SeedData(cfg); err != nil {
			logger.Error("Failed to seed data during initialization", zap.Error(err))
		} else {
			logger.Info("Database seeded with sample data")
		}
	}
}

func handleMigrate(cfg *config.Config) {
	logger.Info("Running database migrations...")

	if err := database.AutoMigrate(); err != nil {
		logger.Fatal("Migration failed", zap.Error(err))
	}

	logger.Info("Database migrations completed successfully")
}

func handleSeed(cfg *config.Config) {
	logger.Info("Seeding database with sample data...")

	if err := database.SeedData(cfg); err != nil {
		logger.Fatal("Seeding failed", zap.Error(err))
	}

	logger.Info("Database seeded successfully")
	fmt.Println("Test users created:")
	fmt.Printf("  Admin:   %s / %s\n", cfg.Admin.Email, cfg.Admin.Password)
	fmt.Println("  Berater: berater@elterngeld-portal.de / berater123")
	fmt.Println("  User:    user@example.com / user123")
}

func startServer(cfg *config.Config) {
	logger.Info("Starting Elterngeld Portal API",
		zap.String("version", "1.0.0"),
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Seed development data if enabled
	if cfg.IsDevelopment() && cfg.Dev.SeedData {
		if err := database.SeedData(cfg); err != nil {
			logger.Error("Failed to seed development data", zap.Error(err))
		}
	}

	// Initialize and start server
	srv := server.New(cfg, logger.Logger)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: srv.Router,

		// Timeouts
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,

		// Security headers
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.String("address", httpServer.Addr),
		)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close database connection
	if err := database.Close(); err != nil {
		logger.Error("Failed to close database connection", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}
