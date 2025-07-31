package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"elterngeld-portal/config"
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

func main() {
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

	logger.Info("Starting Elterngeld Portal API",
		zap.String("version", "1.0.0"),
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Connect to database
	if err := database.Connect(cfg, logger.Logger); err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Seed development data
	if cfg.IsDevelopment() {
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