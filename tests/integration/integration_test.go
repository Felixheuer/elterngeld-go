package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elterngeld-portal/internal/models"
	"elterngeld-portal/internal/server"
	"elterngeld-portal/tests/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoints tests the basic health check endpoints
func TestHealthEndpoints(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	t.Run("health_check", func(t *testing.T) {
		w := client.GET("/health", nil)
		
		assert.Equal(t, http.StatusOK, w.Code)
		testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "elterngeld-portal-api",
		})
	})

	t.Run("readiness_check", func(t *testing.T) {
		w := client.GET("/ready", nil)
		
		assert.Equal(t, http.StatusOK, w.Code)
		testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "ready",
		})
	})
}

// TestUserRegistrationAndAuthentication tests the complete user registration and authentication flow
func TestUserRegistrationAndAuthentication(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	// Since the actual registration endpoint is a placeholder, we'll test the authentication flow
	// by creating a user directly and then testing token validation

	user := testutils.CreateTestUser(t, ctx.DB, models.RoleUser)
	token := testutils.GenerateAuthToken(t, ctx.JWTService, user)

	t.Run("protected_endpoint_without_auth", func(t *testing.T) {
		w := client.GET("/api/v1/auth/me", nil)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Authorization header is required")
	})

	t.Run("protected_endpoint_with_valid_auth", func(t *testing.T) {
		w := client.GET("/api/v1/auth/me", testutils.WithAuth(token))
		
		// Should reach placeholder handler (since endpoint is not implemented)
		assert.Equal(t, http.StatusOK, w.Code)
		testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"message": "Endpoint not yet implemented",
		})
	})

	t.Run("protected_endpoint_with_invalid_token", func(t *testing.T) {
		w := client.GET("/api/v1/auth/me", testutils.WithAuth("invalid-token"))
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Invalid token")
	})
}

// TestRoleBasedAccess tests role-based access control across different endpoints
func TestRoleBasedAccess(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	// Create users with different roles
	admin := testutils.CreateTestUser(t, ctx.DB, models.RoleAdmin)
	berater := testutils.CreateTestUser(t, ctx.DB, models.RoleBerater)
	user := testutils.CreateTestUser(t, ctx.DB, models.RoleUser)

	adminToken := testutils.GenerateAuthToken(t, ctx.JWTService, admin)
	beraterToken := testutils.GenerateAuthToken(t, ctx.JWTService, berater)
	userToken := testutils.GenerateAuthToken(t, ctx.JWTService, user)

	tests := []struct {
		name        string
		endpoint    string
		method      string
		adminOK     bool
		beraterOK   bool
		userOK      bool
	}{
		{
			name:      "admin_stats",
			endpoint:  "/api/v1/admin/stats",
			method:    "GET",
			adminOK:   true,
			beraterOK: false,
			userOK:    false,
		},
		{
			name:      "admin_users",
			endpoint:  "/api/v1/admin/users",
			method:    "GET",
			adminOK:   true,
			beraterOK: false,
			userOK:    false,
		},
		{
			name:      "berater_leads",
			endpoint:  "/api/v1/berater/leads",
			method:    "GET",
			adminOK:   true,
			beraterOK: true,
			userOK:    false,
		},
		{
			name:      "list_users",
			endpoint:  "/api/v1/users",
			method:    "GET",
			adminOK:   true,
			beraterOK: true,
			userOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test admin access
			t.Run("admin_access", func(t *testing.T) {
				w := client.GET(tt.endpoint, testutils.WithAuth(adminToken))
				if tt.adminOK {
					assert.Equal(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusForbidden, w.Code)
				}
			})

			// Test berater access
			t.Run("berater_access", func(t *testing.T) {
				w := client.GET(tt.endpoint, testutils.WithAuth(beraterToken))
				if tt.beraterOK {
					assert.Equal(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusForbidden, w.Code)
				}
			})

			// Test user access
			t.Run("user_access", func(t *testing.T) {
				w := client.GET(tt.endpoint, testutils.WithAuth(userToken))
				if tt.userOK {
					assert.Equal(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusForbidden, w.Code)
				}
			})
		})
	}
}

// TestWebhookAuthentication tests webhook endpoint authentication
func TestWebhookAuthentication(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	// Set webhook secret
	ctx.Config.Stripe.WebhookSecret = "test-webhook-secret"

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	t.Run("webhook_without_api_key", func(t *testing.T) {
		w := client.POST("/api/v1/webhooks/stripe", `{"test": "payload"}`, nil)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "API key is required")
	})

	t.Run("webhook_with_invalid_api_key", func(t *testing.T) {
		headers := map[string]string{
			"X-API-Key": "invalid-key",
		}
		w := client.POST("/api/v1/webhooks/stripe", `{"test": "payload"}`, headers)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Invalid API key")
	})

	t.Run("webhook_with_valid_api_key", func(t *testing.T) {
		headers := map[string]string{
			"X-API-Key": ctx.Config.Stripe.WebhookSecret,
		}
		w := client.POST("/api/v1/webhooks/stripe", `{"test": "payload"}`, headers)
		
		assert.Equal(t, http.StatusOK, w.Code)
		testutils.AssertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"message": "Endpoint not yet implemented",
		})
	})
}

// TestCompleteUserWorkflow tests a complete user workflow from creation to lead management
func TestCompleteUserWorkflow(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	// Create complete test data
	testData := testutils.CreateCompleteTestData(t, ctx.DB)

	// Generate tokens for different users
	adminToken := testutils.GenerateAuthToken(t, ctx.JWTService, testData.Admin)
	beraterToken := testutils.GenerateAuthToken(t, ctx.JWTService, testData.Berater)
	userToken := testutils.GenerateAuthToken(t, ctx.JWTService, testData.User)

	t.Run("user_can_access_own_leads", func(t *testing.T) {
		w := client.GET("/api/v1/leads", testutils.WithAuth(userToken))
		
		// Should reach placeholder (endpoint not implemented)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("berater_can_access_berater_endpoints", func(t *testing.T) {
		w := client.GET("/api/v1/berater/leads", testutils.WithAuth(beraterToken))
		
		// Should reach placeholder (endpoint not implemented)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("admin_can_access_admin_endpoints", func(t *testing.T) {
		w := client.GET("/api/v1/admin/stats", testutils.WithAuth(adminToken))
		
		// Should reach placeholder (endpoint not implemented)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("user_cannot_access_admin_endpoints", func(t *testing.T) {
		w := client.GET("/api/v1/admin/stats", testutils.WithAuth(userToken))
		
		assert.Equal(t, http.StatusForbidden, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusForbidden, "Insufficient permissions")
	})
}

// TestCORSHandling tests CORS header handling
func TestCORSHandling(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	// Configure CORS
	ctx.Config.CORS.Origins = []string{"http://localhost:3000", "https://example.com"}
	ctx.Config.CORS.Credentials = true

	srv := server.New(ctx.Config, ctx.Logger)

	t.Run("cors_allowed_origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		
		srv.Router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("cors_preflight", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/api/v1/auth/login", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		
		srv.Router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	})

	t.Run("cors_not_allowed_origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "http://evil.com")
		
		srv.Router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// TestSecurityHeaders tests that security headers are properly set
func TestSecurityHeaders(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	w := client.GET("/health", nil)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check security headers
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

// TestRequestIDTracking tests that request IDs are properly generated and tracked
func TestRequestIDTracking(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	t.Run("generates_request_id", func(t *testing.T) {
		w := client.GET("/health", nil)
		
		assert.Equal(t, http.StatusOK, w.Code)
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		assert.Len(t, requestID, 36) // UUID length
	})

	t.Run("preserves_existing_request_id", func(t *testing.T) {
		customID := "custom-request-id-123"
		headers := map[string]string{
			"X-Request-ID": customID,
		}
		w := client.GET("/health", headers)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, customID, w.Header().Get("X-Request-ID"))
	})
}

// TestRateLimiting tests rate limiting functionality
func TestRateLimiting(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	// Configure aggressive rate limiting for testing
	ctx.Config.RateLimit.Requests = 2
	ctx.Config.RateLimit.Window = 60

	srv := server.New(ctx.Config, ctx.Logger)

	// Make requests up to the limit
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "127.0.0.1:12345" // Consistent IP for rate limiting
		
		srv.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	
	srv.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	testutils.AssertErrorResponse(t, w, http.StatusTooManyRequests, "Rate limit exceeded")
}

// TestDatabaseIntegration tests that the application properly integrates with the database
func TestDatabaseIntegration(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	// Test that database connection is working by creating and retrieving data
	user := testutils.CreateTestUser(t, ctx.DB, models.RoleUser)
	
	// Verify user was created
	var retrievedUser models.User
	err := ctx.DB.First(&retrievedUser, user.ID).Error
	require.NoError(t, err)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.Role, retrievedUser.Role)

	// Test relationships
	lead := testutils.CreateTestLead(t, ctx.DB, user.ID, nil)
	
	var userWithLeads models.User
	err = ctx.DB.Preload("Leads").First(&userWithLeads, user.ID).Error
	require.NoError(t, err)
	assert.Len(t, userWithLeads.Leads, 1)
	assert.Equal(t, lead.ID, userWithLeads.Leads[0].ID)
}

// TestErrorHandling tests error handling and recovery
func TestErrorHandling(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)

	// Add a route that panics for testing recovery
	srv.Router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	
	srv.Router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	testutils.AssertErrorResponse(t, w, http.StatusInternalServerError, "Internal server error")
	
	// Should include request ID in error response
	var response map[string]interface{}
	testutils.ParseJSONResponse(t, w, &response)
	assert.Contains(t, response, "request_id")
}

// TestConcurrentRequests tests that the server handles concurrent requests properly
func TestConcurrentRequests(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)

	const numRequests = 10
	results := make(chan int, numRequests)

	// Make concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)
			srv.Router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	// Verify all requests succeeded
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		assert.Equal(t, http.StatusOK, statusCode)
	}
}

// TestTokenBlacklisting tests token blacklisting functionality
func TestTokenBlacklisting(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)
	client := testutils.NewTestHTTPClient(srv.Router)

	user := testutils.CreateTestUser(t, ctx.DB, models.RoleUser)
	token := testutils.GenerateAuthToken(t, ctx.JWTService, user)

	t.Run("valid_token_works", func(t *testing.T) {
		w := client.GET("/api/v1/auth/me", testutils.WithAuth(token))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blacklisted_token_rejected", func(t *testing.T) {
		// Blacklist the token
		err := ctx.JWTService.BlacklistToken(token)
		require.NoError(t, err)

		w := client.GET("/api/v1/auth/me", testutils.WithAuth(token))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Token has been revoked")
	})
}

// TestContentTypeHandling tests that the server properly handles different content types
func TestContentTypeHandling(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	srv := server.New(ctx.Config, ctx.Logger)

	t.Run("json_response", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		srv.Router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})
}