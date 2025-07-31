package middleware

import (
	"net/http"

	"elterngeld-portal/internal/models"
	"elterngeld-portal/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
				"code":  "MISSING_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		token := auth.ExtractTokenFromBearer(authHeader)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "INVALID_AUTH_FORMAT",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtService.ValidateTokenWithBlacklist(token)
		if err != nil {
			var code string
			var message string

			switch err {
			case auth.ErrTokenExpired:
				code = "TOKEN_EXPIRED"
				message = "Token has expired"
			case auth.ErrTokenBlacklisted:
				code = "TOKEN_REVOKED"
				message = "Token has been revoked"
			default:
				code = "TOKEN_INVALID"
				message = "Invalid token"
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": message,
				"code":  code,
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("jwt_claims", claims)

		c.Next()
	}
}

// RequireRole ensures the user has the specified role
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found in context",
				"code":  "MISSING_USER_ROLE",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user role type",
				"code":  "INVALID_ROLE_TYPE",
			})
			c.Abort()
			return
		}

		// Check if user has required role
		for _, requiredRole := range roles {
			if role == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
			"code":  "INSUFFICIENT_PERMISSIONS",
		})
		c.Abort()
	}
}

// RequireAdmin ensures the user is an admin
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// RequireBeraterOrAdmin ensures the user is a berater or admin
func RequireBeraterOrAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleBerater, models.RoleAdmin)
}

// RequireOwnershipOrRole checks if user owns the resource or has required role
func RequireOwnershipOrRole(resourceUserIDKey string, roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User ID not found in context",
				"code":  "MISSING_USER_ID",
			})
			c.Abort()
			return
		}

		currentUserID, ok := userID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user ID type",
				"code":  "INVALID_USER_ID_TYPE",
			})
			c.Abort()
			return
		}

		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found in context",
				"code":  "MISSING_USER_ROLE",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user role type",
				"code":  "INVALID_ROLE_TYPE",
			})
			c.Abort()
			return
		}

		// Check if user has required role
		for _, requiredRole := range roles {
			if role == requiredRole {
				c.Next()
				return
			}
		}

		// Check ownership
		resourceUserID, exists := c.Get(resourceUserIDKey)
		if exists {
			if resourceID, ok := resourceUserID.(uuid.UUID); ok && resourceID == currentUserID {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"code":  "ACCESS_DENIED",
		})
		c.Abort()
	}
}

// OptionalAuth middleware that validates token if present but doesn't require it
func OptionalAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token := auth.ExtractTokenFromBearer(authHeader)
		if token == "" {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateTokenWithBlacklist(token)
		if err != nil {
			// Don't abort, just continue without user context
			c.Next()
			return
		}

		// Set user information in context if token is valid
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("jwt_claims", claims)

		c.Next()
	}
}

// GetCurrentUserID extracts the current user ID from context
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	id, ok := userID.(uuid.UUID)
	return id, ok
}

// GetCurrentUserRole extracts the current user role from context
func GetCurrentUserRole(c *gin.Context) (models.UserRole, bool) {
	userRole, exists := c.Get("user_role")
	if !exists {
		return "", false
	}

	role, ok := userRole.(models.UserRole)
	return role, ok
}

// GetCurrentUserEmail extracts the current user email from context
func GetCurrentUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}

	userEmail, ok := email.(string)
	return userEmail, ok
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("user_id")
	return exists
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c *gin.Context) bool {
	role, exists := GetCurrentUserRole(c)
	return exists && role == models.RoleAdmin
}

// IsBerater checks if the current user is a berater
func IsBerater(c *gin.Context) bool {
	role, exists := GetCurrentUserRole(c)
	return exists && role == models.RoleBerater
}

// IsBeraterOrAdmin checks if the current user is a berater or admin
func IsBeraterOrAdmin(c *gin.Context) bool {
	role, exists := GetCurrentUserRole(c)
	return exists && (role == models.RoleBerater || role == models.RoleAdmin)
}

// CanAccessResource checks if user can access a resource based on ownership or role
func CanAccessResource(c *gin.Context, resourceUserID uuid.UUID, allowedRoles ...models.UserRole) bool {
	currentUserID, exists := GetCurrentUserID(c)
	if !exists {
		return false
	}

	// Check ownership
	if currentUserID == resourceUserID {
		return true
	}

	// Check role
	currentRole, exists := GetCurrentUserRole(c)
	if !exists {
		return false
	}

	for _, role := range allowedRoles {
		if currentRole == role {
			return true
		}
	}

	return false
}

// APIKeyMiddleware validates API keys (for webhook endpoints)
func APIKeyMiddleware(validAPIKeys map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Try query parameter
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key is required",
				"code":  "MISSING_API_KEY",
			})
			c.Abort()
			return
		}

		// Validate API key
		keyName, exists := validAPIKeys[apiKey]
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
				"code":  "INVALID_API_KEY",
			})
			c.Abort()
			return
		}

		// Set API key info in context
		c.Set("api_key_name", keyName)
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(allowedOrigins []string, allowCredentials bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Authorization, X-Requested-With, X-API-Key")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}