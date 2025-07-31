package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"elterngeld-portal/internal/models"
	"elterngeld-portal/tests/testutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	user := testutils.CreateTestUser(t, ctx.DB, models.RoleUser)
	token := testutils.GenerateAuthToken(t, ctx.JWTService, user)

	middleware := AuthMiddleware(ctx.JWTService)

	t.Run("valid_token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, user.ID, c.MustGet("user_id"))
		assert.Equal(t, user.Email, c.MustGet("user_email"))
		assert.Equal(t, user.Role, c.MustGet("user_role"))
		assert.NotNil(t, c.MustGet("jwt_claims"))
	})

	t.Run("missing_auth_header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Authorization header is required")
	})

	t.Run("invalid_auth_format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Invalid format")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Invalid authorization header format")
	})

	t.Run("invalid_token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid-token")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Invalid token")
	})

	t.Run("blacklisted_token", func(t *testing.T) {
		// Blacklist the token
		err := ctx.JWTService.BlacklistToken(token)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Token has been revoked")
	})
}

func TestRequireRole(t *testing.T) {
	testutils.SetupGinTestMode()

	tests := []struct {
		name           string
		userRole       models.UserRole
		requiredRoles  []models.UserRole
		shouldPass     bool
		expectedStatus int
	}{
		{"admin_requires_admin", models.RoleAdmin, []models.UserRole{models.RoleAdmin}, true, 0},
		{"berater_requires_admin", models.RoleBerater, []models.UserRole{models.RoleAdmin}, false, http.StatusForbidden},
		{"user_requires_admin", models.RoleUser, []models.UserRole{models.RoleAdmin}, false, http.StatusForbidden},
		{"berater_requires_berater_or_admin", models.RoleBerater, []models.UserRole{models.RoleBerater, models.RoleAdmin}, true, 0},
		{"admin_requires_berater_or_admin", models.RoleAdmin, []models.UserRole{models.RoleBerater, models.RoleAdmin}, true, 0},
		{"user_requires_berater_or_admin", models.RoleUser, []models.UserRole{models.RoleBerater, models.RoleAdmin}, false, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RequireRole(tt.requiredRoles...)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Set("user_role", tt.userRole)

			middleware(c)

			if tt.shouldPass {
				assert.False(t, c.IsAborted())
			} else {
				assert.True(t, c.IsAborted())
				assert.Equal(t, tt.expectedStatus, w.Code)
				testutils.AssertErrorResponse(t, w, tt.expectedStatus, "Insufficient permissions")
			}
		})
	}
}

func TestRequireRole_MissingRole(t *testing.T) {
	testutils.SetupGinTestMode()
	middleware := RequireRole(models.RoleAdmin)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	// Don't set user_role

	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "User role not found in context")
}

func TestRequireRole_InvalidRoleType(t *testing.T) {
	testutils.SetupGinTestMode()
	middleware := RequireRole(models.RoleAdmin)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("user_role", "invalid-role-type") // Wrong type

	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	testutils.AssertErrorResponse(t, w, http.StatusInternalServerError, "Invalid user role type")
}

func TestRequireAdmin(t *testing.T) {
	testutils.SetupGinTestMode()
	middleware := RequireAdmin()

	t.Run("admin_user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set("user_role", models.RoleAdmin)

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("non_admin_user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set("user_role", models.RoleUser)

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRequireBeraterOrAdmin(t *testing.T) {
	testutils.SetupGinTestMode()
	middleware := RequireBeraterOrAdmin()

	tests := []struct {
		name     string
		role     models.UserRole
		shouldPass bool
	}{
		{"admin", models.RoleAdmin, true},
		{"berater", models.RoleBerater, true},
		{"user", models.RoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Set("user_role", tt.role)

			middleware(c)

			if tt.shouldPass {
				assert.False(t, c.IsAborted())
			} else {
				assert.True(t, c.IsAborted())
				assert.Equal(t, http.StatusForbidden, w.Code)
			}
		})
	}
}

func TestRequireOwnershipOrRole(t *testing.T) {
	testutils.SetupGinTestMode()
	middleware := RequireOwnershipOrRole("user_id", models.RoleAdmin)

	userID := uuid.New()
	otherUserID := uuid.New()

	t.Run("owner_access", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set("user_id", userID)
		c.Set("user_role", models.RoleUser)
		c.Params = gin.Params{{Key: "user_id", Value: userID.String()}} // Resource belongs to this user

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("admin_access_to_other_user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set("user_id", userID)
		c.Set("user_role", models.RoleAdmin)
		c.Params = gin.Params{{Key: "user_id", Value: otherUserID.String()}} // Resource belongs to other user

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("user_access_to_other_user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set("user_id", userID)
		c.Set("user_role", models.RoleUser)
		c.Params = gin.Params{{Key: "user_id", Value: otherUserID.String()}} // Resource belongs to other user

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusForbidden, "Access denied")
	})
}

func TestOptionalAuth(t *testing.T) {
	testutils.SetupGinTestMode()
	ctx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(ctx)

	user := testutils.CreateTestUser(t, ctx.DB, models.RoleUser)
	token := testutils.GenerateAuthToken(t, ctx.JWTService, user)

	middleware := OptionalAuth(ctx.JWTService)

	t.Run("with_valid_token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, user.ID, c.MustGet("user_id"))
		assert.Equal(t, user.Email, c.MustGet("user_email"))
		assert.Equal(t, user.Role, c.MustGet("user_role"))
	})

	t.Run("without_auth_header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		middleware(c)

		assert.False(t, c.IsAborted())
		// Should not set any user context
		_, exists := c.Get("user_id")
		assert.False(t, exists)
	})

	t.Run("with_invalid_token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid-token")

		middleware(c)

		assert.False(t, c.IsAborted())
		// Should not set any user context
		_, exists := c.Get("user_id")
		assert.False(t, exists)
	})
}

func TestGetCurrentUserID(t *testing.T) {
	testutils.SetupGinTestMode()
	userID := uuid.New()

	t.Run("with_user_id", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", userID)

		id, ok := GetCurrentUserID(c)
		assert.True(t, ok)
		assert.Equal(t, userID, id)
	})

	t.Run("without_user_id", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		id, ok := GetCurrentUserID(c)
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, id)
	})

	t.Run("with_invalid_type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", "invalid-uuid-type")

		id, ok := GetCurrentUserID(c)
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, id)
	})
}

func TestGetCurrentUserRole(t *testing.T) {
	testutils.SetupGinTestMode()

	t.Run("with_user_role", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_role", models.RoleAdmin)

		role, ok := GetCurrentUserRole(c)
		assert.True(t, ok)
		assert.Equal(t, models.RoleAdmin, role)
	})

	t.Run("without_user_role", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		role, ok := GetCurrentUserRole(c)
		assert.False(t, ok)
		assert.Equal(t, models.UserRole(""), role)
	})

	t.Run("with_invalid_type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_role", "invalid-role-type")

		role, ok := GetCurrentUserRole(c)
		assert.False(t, ok)
		assert.Equal(t, models.UserRole(""), role)
	})
}

func TestGetCurrentUserEmail(t *testing.T) {
	testutils.SetupGinTestMode()

	t.Run("with_user_email", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_email", "test@example.com")

		email, ok := GetCurrentUserEmail(c)
		assert.True(t, ok)
		assert.Equal(t, "test@example.com", email)
	})

	t.Run("without_user_email", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		email, ok := GetCurrentUserEmail(c)
		assert.False(t, ok)
		assert.Equal(t, "", email)
	})

	t.Run("with_invalid_type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_email", 123)

		email, ok := GetCurrentUserEmail(c)
		assert.False(t, ok)
		assert.Equal(t, "", email)
	})
}

func TestIsAuthenticated(t *testing.T) {
	testutils.SetupGinTestMode()

	t.Run("authenticated", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", uuid.New())

		assert.True(t, IsAuthenticated(c))
	})

	t.Run("not_authenticated", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		assert.False(t, IsAuthenticated(c))
	})
}

func TestIsAdmin(t *testing.T) {
	testutils.SetupGinTestMode()

	t.Run("admin_user", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_role", models.RoleAdmin)

		assert.True(t, IsAdmin(c))
	})

	t.Run("non_admin_user", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_role", models.RoleUser)

		assert.False(t, IsAdmin(c))
	})

	t.Run("no_role", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		assert.False(t, IsAdmin(c))
	})
}

func TestIsBerater(t *testing.T) {
	testutils.SetupGinTestMode()

	t.Run("berater_user", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_role", models.RoleBerater)

		assert.True(t, IsBerater(c))
	})

	t.Run("non_berater_user", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_role", models.RoleUser)

		assert.False(t, IsBerater(c))
	})
}

func TestIsBeraterOrAdmin(t *testing.T) {
	testutils.SetupGinTestMode()

	tests := []struct {
		name     string
		role     models.UserRole
		expected bool
	}{
		{"admin", models.RoleAdmin, true},
		{"berater", models.RoleBerater, true},
		{"user", models.RoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("user_role", tt.role)

			assert.Equal(t, tt.expected, IsBeraterOrAdmin(c))
		})
	}
}

func TestCanAccessResource(t *testing.T) {
	testutils.SetupGinTestMode()

	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name         string
		currentRole  models.UserRole
		resourceUser uuid.UUID
		allowedRoles []models.UserRole
		expected     bool
	}{
		{"owner_access", models.RoleUser, userID, []models.UserRole{models.RoleAdmin}, true},
		{"admin_access", models.RoleAdmin, otherUserID, []models.UserRole{models.RoleAdmin}, true},
		{"berater_access", models.RoleBerater, otherUserID, []models.UserRole{models.RoleBerater}, true},
		{"user_no_access", models.RoleUser, otherUserID, []models.UserRole{models.RoleAdmin}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("user_id", userID)
			c.Set("user_role", tt.currentRole)

			result := CanAccessResource(c, tt.resourceUser, tt.allowedRoles...)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("no_user_context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		result := CanAccessResource(c, userID, models.RoleAdmin)
		assert.False(t, result)
	})
}

func TestAPIKeyMiddleware(t *testing.T) {
	testutils.SetupGinTestMode()

	validAPIKeys := map[string]string{
		"test-api-key-123": "stripe",
		"admin-key-456":    "admin",
	}

	middleware := APIKeyMiddleware(validAPIKeys)

	t.Run("valid_api_key_header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/webhook", nil)
		c.Request.Header.Set("X-API-Key", "test-api-key-123")

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, "stripe", c.MustGet("api_key_name"))
	})

	t.Run("valid_api_key_query", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/webhook?api_key=admin-key-456", nil)

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, "admin", c.MustGet("api_key_name"))
	})

	t.Run("missing_api_key", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/webhook", nil)

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "API key is required")
	})

	t.Run("invalid_api_key", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/webhook", nil)
		c.Request.Header.Set("X-API-Key", "invalid-key")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		testutils.AssertErrorResponse(t, w, http.StatusUnauthorized, "Invalid API key")
	})
}

func TestCORSMiddleware(t *testing.T) {
	testutils.SetupGinTestMode()

	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	middleware := CORSMiddleware(allowedOrigins, true)

	t.Run("allowed_origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Origin", "http://localhost:3000")

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})

	t.Run("wildcard_origin", func(t *testing.T) {
		wildcardMiddleware := CORSMiddleware([]string{"*"}, false)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Origin", "http://any-origin.com")

		wildcardMiddleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, "http://any-origin.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("not_allowed_origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Origin", "http://evil.com")

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("options_preflight", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("OPTIONS", "/test", nil)
		c.Request.Header.Set("Origin", "http://localhost:3000")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	})
}