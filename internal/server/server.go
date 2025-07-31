package server

import (
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/middleware"
	"elterngeld-portal/pkg/auth"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	Router     *gin.Engine
	config     *config.Config
	logger     *zap.Logger
	jwtService *auth.JWTService
}

// New creates a new server instance
func New(cfg *config.Config, logger *zap.Logger) *Server {
	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.New()

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg)

	server := &Server{
		Router:     router,
		config:     cfg,
		logger:     logger,
		jwtService: jwtService,
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// Basic middleware
	s.Router.Use(middleware.RequestIDMiddleware())
	s.Router.Use(middleware.RecoveryMiddleware(s.logger))
	s.Router.Use(middleware.SecurityHeadersMiddleware())

	// CORS middleware
	s.Router.Use(middleware.CORSMiddleware(
		s.config.CORS.Origins,
		s.config.CORS.Credentials,
	))

	// Rate limiting middleware
	rateLimiter := middleware.NewRateLimit(
		s.config.RateLimit.Requests,
		time.Duration(s.config.RateLimit.Window)*time.Second,
	)
	s.Router.Use(middleware.RateLimitMiddleware(rateLimiter, s.logger))

	// Logging middleware
	if s.config.IsDevelopment() {
		s.Router.Use(middleware.DetailedLoggingMiddleware(s.logger, false, false))
	} else {
		s.Router.Use(middleware.LoggingMiddleware(s.logger))
	}
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.Router.GET("/health", s.healthCheck)
	s.Router.GET("/ready", s.readinessCheck)

	// Swagger documentation
	if s.config.IsDevelopment() {
		s.Router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API v1 routes
	v1 := s.Router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		public := v1.Group("")
		{
			// Authentication routes
			auth := public.Group("/auth")
			{
				auth.POST("/register", s.placeholder("Register"))
				auth.POST("/login", s.placeholder("Login"))
				auth.POST("/refresh", s.placeholder("Refresh Token"))
				auth.POST("/forgot-password", s.placeholder("Forgot Password"))
				auth.POST("/reset-password", s.placeholder("Reset Password"))
			}

			// Webhook routes (with API key authentication)
			webhooks := public.Group("/webhooks")
			webhooks.Use(middleware.APIKeyMiddleware(map[string]string{
				s.config.Stripe.WebhookSecret: "stripe",
			}))
			{
				webhooks.POST("/stripe", s.placeholder("Stripe Webhook"))
			}
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(s.jwtService))
		{
			// Authentication routes for authenticated users
			auth := protected.Group("/auth")
			{
				auth.POST("/logout", s.placeholder("Logout"))
				auth.GET("/me", s.placeholder("Get Current User"))
				auth.PUT("/me", s.placeholder("Update Current User"))
				auth.POST("/change-password", s.placeholder("Change Password"))
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("", middleware.RequireBeraterOrAdmin(), s.placeholder("List Users"))
				users.GET("/:id", middleware.RequireOwnershipOrRole("user_id", "berater", "admin"), s.placeholder("Get User"))
				users.PUT("/:id", middleware.RequireOwnershipOrRole("user_id", "admin"), s.placeholder("Update User"))
				users.DELETE("/:id", middleware.RequireAdmin(), s.placeholder("Delete User"))
			}

			// Lead routes
			leads := protected.Group("/leads")
			{
				leads.GET("", s.placeholder("List Leads"))
				leads.POST("", s.placeholder("Create Lead"))
				leads.GET("/:id", s.placeholder("Get Lead"))
				leads.PUT("/:id", s.placeholder("Update Lead"))
				leads.DELETE("/:id", s.placeholder("Delete Lead"))
				leads.PATCH("/:id/status", s.placeholder("Update Lead Status"))
				leads.POST("/:id/assign", middleware.RequireBeraterOrAdmin(), s.placeholder("Assign Lead"))

				// Lead comments
				leads.GET("/:id/comments", s.placeholder("List Lead Comments"))
				leads.POST("/:id/comments", s.placeholder("Create Lead Comment"))
				leads.PUT("/comments/:commentId", s.placeholder("Update Lead Comment"))
				leads.DELETE("/comments/:commentId", s.placeholder("Delete Lead Comment"))
			}

			// Document routes
			documents := protected.Group("/documents")
			{
				documents.GET("", s.placeholder("List Documents"))
				documents.POST("", s.placeholder("Upload Document"))
				documents.GET("/:id", s.placeholder("Get Document"))
				documents.PUT("/:id", s.placeholder("Update Document"))
				documents.DELETE("/:id", s.placeholder("Delete Document"))
				documents.GET("/:id/download", s.placeholder("Download Document"))
			}

			// Payment routes
			payments := protected.Group("/payments")
			{
				payments.GET("", s.placeholder("List Payments"))
				payments.POST("/checkout", s.placeholder("Create Stripe Checkout"))
				payments.GET("/:id", s.placeholder("Get Payment"))
				payments.POST("/:id/refund", middleware.RequireBeraterOrAdmin(), s.placeholder("Refund Payment"))
			}

			// Activity routes
			activities := protected.Group("/activities")
			{
				activities.GET("", s.placeholder("List Activities"))
				activities.GET("/:id", s.placeholder("Get Activity"))
			}

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/stats", s.placeholder("Get Admin Stats"))
				admin.GET("/users", s.placeholder("Admin List Users"))
				admin.POST("/users", s.placeholder("Admin Create User"))
				admin.PUT("/users/:id/role", s.placeholder("Admin Change User Role"))
				admin.PUT("/users/:id/status", s.placeholder("Admin Change User Status"))

				admin.GET("/leads", s.placeholder("Admin List Leads"))
				admin.GET("/payments", s.placeholder("Admin List Payments"))
				admin.GET("/activities", s.placeholder("Admin List Activities"))
				admin.GET("/system", s.placeholder("System Information"))
			}

			// Berater routes
			berater := protected.Group("/berater")
			berater.Use(middleware.RequireBeraterOrAdmin())
			{
				berater.GET("/leads", s.placeholder("Berater List Assigned Leads"))
				berater.GET("/stats", s.placeholder("Berater Stats"))
			}
		}
	}

	// Payment result pages (public, for Stripe redirects)
	s.Router.GET("/payment/success", s.placeholder("Payment Success Page"))
	s.Router.GET("/payment/cancel", s.placeholder("Payment Cancel Page"))

	// Static file serving (for uploaded documents, only in development)
	if s.config.IsDevelopment() && !s.config.S3.UseS3 {
		s.Router.Static("/uploads", s.config.Upload.Path)
	}
}

// placeholder creates a placeholder handler for unimplemented endpoints
func (s *Server) placeholder(description string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":     "Endpoint not yet implemented",
			"description": description,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
		})
	}
}

// healthCheck handles health check requests
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"service":   "elterngeld-portal-api",
	})
}

// readinessCheck handles readiness check requests
// @Summary Readiness check
// @Description Check if the service is ready to serve requests
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /ready [get]
func (s *Server) readinessCheck(c *gin.Context) {
	// Check database connection
	if err := s.checkDatabaseHealth(); err != nil {
		s.logger.Error("Database health check failed", zap.Error(err))
		c.JSON(503, gin.H{
			"status":    "not ready",
			"timestamp": time.Now().UTC(),
			"error":     "Database connection failed",
		})
		return
	}

	c.JSON(200, gin.H{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"service":   "elterngeld-portal-api",
		"checks": gin.H{
			"database": "healthy",
		},
	})
}

// checkDatabaseHealth checks if the database is accessible
func (s *Server) checkDatabaseHealth() error {
	// Import here to avoid circular dependency
	// In a real implementation, you'd inject the database dependency
	// For now, we'll just return nil to indicate healthy
	return nil
}
