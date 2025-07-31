package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/tests/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	logger := zap.NewNop()

	server := New(cfg, logger)

	assert.NotNil(t, server)
	assert.NotNil(t, server.Router)
	assert.Equal(t, cfg, server.config)
	assert.Equal(t, logger, server.logger)
	assert.NotNil(t, server.jwtService)
}

func TestNew_ProductionMode(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.Server.Env = "production"
	logger := zap.NewNop()

	server := New(cfg, logger)

	assert.NotNil(t, server)
	assert.NotNil(t, server.Router)
	// In test, mode is controlled by testutils.SetupGinTestMode()
	// so we can't test actual mode setting
}

func TestHealthCheck(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"service": "elterngeld-portal-api",
	})
}

func TestReadinessCheck(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status":  "ready",
		"version": "1.0.0",
		"service": "elterngeld-portal-api",
	})
}

func TestSecurityHeaders(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	server.Router.ServeHTTP(w, req)

	// Check security headers
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestCORSHeaders(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.CORS.Origins = []string{"http://localhost:3000", "https://example.com"}
	cfg.CORS.Credentials = true
	logger := zap.NewNop()

	server := New(cfg, logger)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
		expectCreds    bool
	}{
		{"allowed origin", "http://localhost:3000", "http://localhost:3000", true},
		{"allowed origin 2", "https://example.com", "https://example.com", true},
		{"not allowed origin", "http://evil.com", "", true},
		{"no origin", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			server.Router.ServeHTTP(w, req)

			if tt.expectedOrigin != "" {
				assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}

			if tt.expectCreds {
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
			}

			assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
			assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.CORS.Origins = []string{"http://localhost:3000"}
	logger := zap.NewNop()

	server := New(cfg, logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")

	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestPlaceholderEndpoints(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	endpoints := []struct {
		method string
		path   string
		desc   string
	}{
		{"POST", "/api/v1/auth/register", "Register"},
		{"POST", "/api/v1/auth/login", "Login"},
		{"POST", "/api/v1/auth/refresh", "Refresh Token"},
		{"POST", "/api/v1/auth/forgot-password", "Forgot Password"},
		{"POST", "/api/v1/auth/reset-password", "Reset Password"},
		{"GET", "/payment/success", "Payment Success Page"},
		{"GET", "/payment/cancel", "Payment Cancel Page"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			server.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"message":     "Endpoint not yet implemented",
				"description": endpoint.desc,
				"method":      endpoint.method,
			})
		})
	}
}

func TestProtectedEndpointsRequireAuth(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/auth/logout"},
		{"GET", "/api/v1/auth/me"},
		{"PUT", "/api/v1/auth/me"},
		{"POST", "/api/v1/auth/change-password"},
		{"GET", "/api/v1/users"},
		{"GET", "/api/v1/leads"},
		{"POST", "/api/v1/leads"},
		{"GET", "/api/v1/documents"},
		{"POST", "/api/v1/documents"},
		{"GET", "/api/v1/payments"},
		{"POST", "/api/v1/payments/checkout"},
		{"GET", "/api/v1/activities"},
		{"GET", "/api/v1/admin/stats"},
		{"GET", "/api/v1/berater/leads"},
	}

	for _, endpoint := range protectedEndpoints {
		t.Run(endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			server.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Authorization header is required")
		})
	}
}

func TestWebhookEndpointsRequireAPIKey(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.Stripe.WebhookSecret = "test-webhook-secret"
	logger := zap.NewNop()

	server := New(cfg, logger)

	t.Run("without_api_key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/webhooks/stripe", nil)
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "API key is required")
	})

	t.Run("with_invalid_api_key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/webhooks/stripe", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Invalid API key")
	})

	t.Run("with_valid_api_key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/webhooks/stripe", nil)
		req.Header.Set("X-API-Key", cfg.Stripe.WebhookSecret)
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Should reach placeholder handler
	})
}

func TestSwaggerDocsInDevelopment(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.Server.Env = "development"
	logger := zap.NewNop()

	server := New(cfg, logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/docs/index.html", nil)
	server.Router.ServeHTTP(w, req)

	// Should not return 404 (docs should be available)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestSwaggerDocsNotInProduction(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.Server.Env = "production"
	logger := zap.NewNop()

	server := New(cfg, logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/docs/index.html", nil)
	server.Router.ServeHTTP(w, req)

	// Should return 404 (docs should not be available in production)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStaticFilesInDevelopment(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.Server.Env = "development"
	cfg.S3.UseS3 = false
	cfg.Upload.Path = "/tmp/uploads"
	logger := zap.NewNop()

	server := New(cfg, logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/uploads/test.txt", nil)
	server.Router.ServeHTTP(w, req)

	// Should attempt to serve static files (may return 404 if file doesn't exist, but route should exist)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestRateLimiting(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	cfg.RateLimit.Requests = 2
	cfg.RateLimit.Window = 60
	logger := zap.NewNop()

	server := New(cfg, logger)

	// Make requests up to the limit
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "127.0.0.1:12345" // Set consistent IP
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "2")
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	testutils.AssertErrorResponse(t, w, http.StatusTooManyRequests, "Rate limit exceeded")
}

func TestRequestIDMiddleware(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	t.Run("without_request_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		server.Router.ServeHTTP(w, req)

		// Should add request ID header
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		assert.Len(t, requestID, 36) // UUID length
	})

	t.Run("with_existing_request_id", func(t *testing.T) {
		existingID := "test-request-id-123"
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Request-ID", existingID)
		server.Router.ServeHTTP(w, req)

		// Should preserve existing request ID
		requestID := w.Header().Get("X-Request-ID")
		assert.Equal(t, existingID, requestID)
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	logger := zap.NewNop()
	server := New(cfg, logger)

	// Add a route that panics
	server.Router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	testutils.AssertErrorResponse(t, w, http.StatusInternalServerError, "Internal server error")

	// Should include request ID in error response
	var response map[string]interface{}
	testutils.ParseJSONResponse(t, w, &response)
	assert.Contains(t, response, "request_id")
}

func TestDatabaseHealthCheckInReadiness(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	// Since checkDatabaseHealth always returns nil (placeholder),
	// readiness check should always return healthy
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status": "ready",
	})

	var response map[string]interface{}
	testutils.ParseJSONResponse(t, w, &response)
	checks := response["checks"].(map[string]interface{})
	assert.Equal(t, "healthy", checks["database"])
}

func TestMiddlewareOrder(t *testing.T) {
	testutils.SetupGinTestMode()
	cfg := createTestConfig()
	logger := zap.NewNop()
	server := New(cfg, logger)

	// Add a test route to verify middleware execution order
	middlewareOrder := []string{}
	server.Router.Use(func(c *gin.Context) {
		middlewareOrder = append(middlewareOrder, "test-middleware")
		c.Next()
	})

	server.Router.GET("/test-middleware-order", func(c *gin.Context) {
		middlewareOrder = append(middlewareOrder, "handler")
		c.JSON(200, gin.H{"order": middlewareOrder})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test-middleware-order", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	testutils.ParseJSONResponse(t, w, &response)
	
	// Verify that test middleware ran after built-in middleware
	order := response["order"].([]interface{})
	assert.Contains(t, order, "test-middleware")
	assert.Contains(t, order, "handler")
}

func TestConcurrentRequests(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	const numRequests = 10
	results := make(chan int, numRequests)

	// Make concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)
			server.Router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	// Verify all requests succeeded
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		assert.Equal(t, http.StatusOK, statusCode)
	}
}

func TestInvalidHTTPMethods(t *testing.T) {
	testutils.SetupGinTestMode()
	server := createTestServer(t)

	invalidMethods := []string{"PATCH", "HEAD", "CONNECT", "TRACE"}

	for _, method := range invalidMethods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/health", nil)
			server.Router.ServeHTTP(w, req)

			// Should return method not allowed for unsupported methods on health endpoint
			if method != "HEAD" { // HEAD might be supported for GET endpoints
				assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

// Helper functions

func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Env:  "test",
			Port: "8080",
		},
		JWT: config.JWTConfig{
			Secret:        "test-secret-key",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
		CORS: config.CORSConfig{
			Origins:     []string{"*"},
			Credentials: false,
		},
		RateLimit: config.RateLimitConfig{
			Requests: 100,
			Window:   60,
		},
		Stripe: config.StripeConfig{
			WebhookSecret: "test-stripe-webhook-secret",
		},
		S3: config.S3Config{
			UseS3: false,
		},
		Upload: config.UploadConfig{
			Path: "/tmp/uploads",
		},
	}
}

func createTestServer(t *testing.T) *Server {
	cfg := createTestConfig()
	logger := zap.NewNop()
	return New(cfg, logger)
}