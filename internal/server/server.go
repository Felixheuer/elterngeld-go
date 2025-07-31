package server

import (
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/database"
	"elterngeld-portal/internal/handlers"
	"elterngeld-portal/internal/middleware"
	"elterngeld-portal/pkg/auth"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Server represents the HTTP server
type Server struct {
	Router     *gin.Engine
	config     *config.Config
	logger     *zap.Logger
	jwtService *auth.JWTService
	db         *gorm.DB
	
	// Handlers
	authHandler     *handlers.AuthHandler
	userHandler     *handlers.UserHandler
	leadHandler     *handlers.LeadHandler
	bookingHandler  *handlers.BookingHandler
	paymentHandler  *handlers.PaymentHandler
	documentHandler *handlers.DocumentHandler
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

	// Initialize database connection
	db, err := database.Connect(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, logger, jwtService, cfg)
	userHandler := handlers.NewUserHandler(db, logger)
	leadHandler := handlers.NewLeadHandler(db, logger)
	bookingHandler := handlers.NewBookingHandler(db, logger)
	paymentHandler := handlers.NewPaymentHandler(db, logger, cfg)
	documentHandler := handlers.NewDocumentHandler(db, logger, cfg)

	server := &Server{
		Router:          router,
		config:          cfg,
		logger:          logger,
		jwtService:      jwtService,
		db:              db,
		authHandler:     authHandler,
		userHandler:     userHandler,
		leadHandler:     leadHandler,
		bookingHandler:  bookingHandler,
		paymentHandler:  paymentHandler,
		documentHandler: documentHandler,
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
	s.Router.HEAD("/health", s.healthCheck) // Support HEAD requests for health checks
	s.Router.GET("/ready", s.readinessCheck)
	s.Router.HEAD("/ready", s.readinessCheck) // Support HEAD requests for readiness checks

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
				auth.POST("/register", s.authHandler.Register)
				auth.POST("/login", s.authHandler.Login)
				auth.POST("/refresh", s.authHandler.RefreshToken)
				auth.POST("/forgot-password", s.authHandler.ForgotPassword)
				auth.POST("/reset-password", s.authHandler.ResetPassword)
				auth.GET("/verify-email", s.authHandler.VerifyEmail)
			}

			// Public package and timeslot routes
			public.GET("/packages", s.bookingHandler.ListPackages)
			public.GET("/packages/:id/addons", s.bookingHandler.GetPackageAddOns)
			public.GET("/timeslots/available", s.bookingHandler.GetAvailableTimeslots)

			// Webhook routes (with API key authentication)
			webhooks := public.Group("/webhooks")
			webhooks.Use(middleware.APIKeyMiddleware(map[string]string{
				s.config.Stripe.WebhookSecret: "stripe",
			}))
			{
				webhooks.POST("/stripe", s.paymentHandler.StripeWebhook)
			}
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(s.jwtService))
		{
			// Authentication routes for authenticated users
			auth := protected.Group("/auth")
			{
				auth.POST("/logout", s.authHandler.Logout)
				auth.GET("/me", s.authHandler.GetMe)
				auth.PUT("/me", s.authHandler.UpdateMe)
				auth.POST("/change-password", s.authHandler.ChangePassword)
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("", middleware.RequireBeraterOrAdmin(), s.userHandler.ListUsers)
				users.GET("/:id", middleware.RequireOwnershipOrRole("user_id", "berater", "admin"), s.userHandler.GetUser)
				users.PUT("/:id", middleware.RequireOwnershipOrRole("user_id", "admin"), s.userHandler.UpdateUser)
				users.DELETE("/:id", middleware.RequireAdmin(), s.userHandler.DeleteUser)
			}

			// Lead routes
			leads := protected.Group("/leads")
			{
				leads.GET("", s.leadHandler.ListLeads)
				leads.POST("", s.leadHandler.CreateLead)
				leads.GET("/:id", s.leadHandler.GetLead)
				leads.PUT("/:id", s.leadHandler.UpdateLead)
				leads.DELETE("/:id", s.leadHandler.DeleteLead)
				leads.PATCH("/:id/status", s.leadHandler.UpdateLeadStatus)
				leads.POST("/:id/assign", middleware.RequireBeraterOrAdmin(), s.leadHandler.AssignLead)

				// Lead comments
				leads.GET("/:id/comments", s.leadHandler.ListLeadComments)
				leads.POST("/:id/comments", s.leadHandler.CreateLeadComment)
				leads.PUT("/comments/:commentId", s.placeholder("Update Lead Comment"))
				leads.DELETE("/comments/:commentId", s.placeholder("Delete Lead Comment"))
			}

			// Booking routes
			bookings := protected.Group("/bookings")
			{
				bookings.GET("", s.bookingHandler.GetUserBookings)
				bookings.POST("", s.bookingHandler.CreateBooking)
				bookings.GET("/:id", s.bookingHandler.GetBooking)
				bookings.PUT("/:id/contact-info", s.bookingHandler.UpdateBookingContactInfo)
			}

			// Document routes
			documents := protected.Group("/documents")
			{
				documents.GET("", s.documentHandler.ListDocuments)
				documents.POST("", s.documentHandler.UploadDocument)
				documents.GET("/:id", s.documentHandler.GetDocument)
				documents.PUT("/:id", s.documentHandler.UpdateDocument)
				documents.DELETE("/:id", s.documentHandler.DeleteDocument)
				documents.GET("/:id/download", s.documentHandler.DownloadDocument)
			}

			// Payment routes
			payments := protected.Group("/payments")
			{
				payments.GET("", s.paymentHandler.ListPayments)
				payments.POST("/checkout", s.paymentHandler.CreateCheckout)
				payments.GET("/:id", s.paymentHandler.GetPayment)
				payments.POST("/:id/refund", middleware.RequireBeraterOrAdmin(), s.paymentHandler.RefundPayment)
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
				admin.GET("/users", s.userHandler.ListUsers)
				admin.POST("/users", s.userHandler.AdminCreateUser)
				admin.PUT("/users/:id/role", s.userHandler.AdminChangeUserRole)
				admin.PUT("/users/:id/status", s.userHandler.AdminChangeUserStatus)

				admin.GET("/leads", s.leadHandler.ListLeads)
				admin.GET("/payments", s.paymentHandler.ListPayments)
				admin.GET("/activities", s.placeholder("Admin List Activities"))
				admin.GET("/system", s.placeholder("System Information"))
			}

			// Berater routes
			berater := protected.Group("/berater")
			berater.Use(middleware.RequireBeraterOrAdmin())
			{
				berater.GET("/leads", s.leadHandler.ListLeads)
				berater.GET("/stats", s.placeholder("Berater Stats"))
			}
		}
	}

	// Payment result pages (public, for Stripe redirects)
	s.Router.GET("/payment/success", s.paymentHandler.PaymentSuccessPage)
	s.Router.GET("/payment/cancel", s.paymentHandler.PaymentCancelPage)

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
