package auth

import (
	"testing"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTService(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}

	service := NewJWTService(cfg)

	assert.NotNil(t, service)
	assert.Equal(t, []byte("test-secret-key"), service.secretKey)
	assert.Equal(t, "elterngeld-portal", service.issuer)
	assert.Equal(t, 15*time.Minute, service.accessTTL)
	assert.Equal(t, 7*24*time.Hour, service.refreshTTL)
}

func TestGenerateTokenPair(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	tokenPair, err := service.GenerateTokenPair(user)

	require.NoError(t, err)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Equal(t, "Bearer", tokenPair.TokenType)
	assert.Equal(t, int64(900), tokenPair.ExpiresIn) // 15 minutes
}

func TestValidateAccessToken(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	// Generate a token
	tokenPair, err := service.GenerateTokenPair(user)
	require.NoError(t, err)

	// Validate the token
	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role, claims.Role)
	assert.Equal(t, "elterngeld-portal", claims.RegisteredClaims.Issuer)
	assert.Equal(t, user.ID.String(), claims.RegisteredClaims.Subject)
	assert.Contains(t, claims.RegisteredClaims.Audience, "elterngeld-portal-api")
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	service := createTestJWTService()

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid token", "invalid.token.here"},
		{"malformed token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateAccessToken(tt.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create service with very short expiry
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key",
			AccessExpiry:  1 * time.Millisecond,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}
	service := NewJWTService(cfg)
	user := createTestUser()

	// Generate token
	tokenPair, err := service.GenerateTokenPair(user)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(2 * time.Millisecond)

	// Try to validate expired token
	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_WrongSigningMethod(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	// Create token with wrong signing method
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "elterngeld-portal",
			Subject:   user.ID.String(),
			Audience:  []string{"elterngeld-portal-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	// Use RS256 instead of HS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, _ := token.SignedString([]byte("fake-key"))

	validatedClaims, err := service.ValidateAccessToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
}

func TestExtractTokenFromBearer(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid bearer token", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"no bearer prefix", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", ""},
		{"empty header", "", ""},
		{"only bearer", "Bearer", ""},
		{"bearer with space only", "Bearer ", ""},
		{"wrong prefix", "Token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTokenFromBearer(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	// Test valid token
	tokenPair, err := service.GenerateTokenPair(user)
	require.NoError(t, err)

	assert.False(t, service.IsTokenExpired(tokenPair.AccessToken))

	// Test expired token
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key",
			AccessExpiry:  1 * time.Millisecond,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}
	expiredService := NewJWTService(cfg)
	expiredTokenPair, err := expiredService.GenerateTokenPair(user)
	require.NoError(t, err)

	time.Sleep(2 * time.Millisecond)
	assert.True(t, expiredService.IsTokenExpired(expiredTokenPair.AccessToken))

	// Test invalid token
	assert.True(t, service.IsTokenExpired("invalid-token"))
}

func TestGetTokenClaims(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	tokenPair, err := service.GenerateTokenPair(user)
	require.NoError(t, err)

	claims, err := service.GetTokenClaims(tokenPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role, claims.Role)
}

func TestRefreshTokenExpiry(t *testing.T) {
	service := createTestJWTService()
	
	before := time.Now()
	expiry := service.RefreshTokenExpiry()
	after := time.Now()

	expectedDuration := 7 * 24 * time.Hour
	assert.True(t, expiry.After(before.Add(expectedDuration-time.Second)))
	assert.True(t, expiry.Before(after.Add(expectedDuration+time.Second)))
}

func TestTokenBlacklist(t *testing.T) {
	blacklist := NewTokenBlacklist()
	tokenID := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)

	// Initially not blacklisted
	assert.False(t, blacklist.IsBlacklisted(tokenID))

	// Add to blacklist
	blacklist.Add(tokenID, expiresAt)
	assert.True(t, blacklist.IsBlacklisted(tokenID))

	// Test with expired token
	expiredTokenID := uuid.New().String()
	expiredAt := time.Now().Add(-1 * time.Hour)
	blacklist.Add(expiredTokenID, expiredAt)
	assert.False(t, blacklist.IsBlacklisted(expiredTokenID))
}

func TestTokenBlacklist_Cleanup(t *testing.T) {
	blacklist := NewTokenBlacklist()
	
	// Add expired token
	expiredTokenID := uuid.New().String()
	blacklist.Add(expiredTokenID, time.Now().Add(-1*time.Hour))
	
	// Add valid token
	validTokenID := uuid.New().String()
	blacklist.Add(validTokenID, time.Now().Add(1*time.Hour))

	// Before cleanup
	assert.Len(t, blacklist.tokens, 2)

	// After cleanup
	blacklist.Cleanup()
	assert.Len(t, blacklist.tokens, 1)
	assert.True(t, blacklist.IsBlacklisted(validTokenID))
	assert.False(t, blacklist.IsBlacklisted(expiredTokenID))
}

func TestValidateTokenWithBlacklist(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	tokenPair, err := service.GenerateTokenPair(user)
	require.NoError(t, err)

	// Valid token
	claims, err := service.ValidateTokenWithBlacklist(tokenPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)

	// Blacklist the token
	err = service.BlacklistToken(tokenPair.AccessToken)
	require.NoError(t, err)

	// Should now fail validation
	claims, err = service.ValidateTokenWithBlacklist(tokenPair.AccessToken)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenBlacklisted, err)
	assert.Nil(t, claims)

	// Invalid token
	claims, err = service.ValidateTokenWithBlacklist("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, ErrTokenInvalid, err)
	assert.Nil(t, claims)
}

func TestBlacklistToken(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	tokenPair, err := service.GenerateTokenPair(user)
	require.NoError(t, err)

	// Blacklist valid token
	err = service.BlacklistToken(tokenPair.AccessToken)
	assert.NoError(t, err)

	// Blacklist invalid token
	err = service.BlacklistToken("invalid-token")
	assert.Error(t, err)
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{"token expired", ErrTokenExpired, "Token has expired"},
		{"token invalid", ErrTokenInvalid, "Invalid token"},
		{"token blacklisted", ErrTokenBlacklisted, "Token has been revoked"},
		{"token malformed", ErrTokenMalformed, "Token is malformed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestConcurrentTokenOperations(t *testing.T) {
	service := createTestJWTService()
	user := createTestUser()

	// Test concurrent token generation
	tokens := make(chan string, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			tokenPair, err := service.GenerateTokenPair(user)
			if err != nil {
				errors <- err
				return
			}
			tokens <- tokenPair.AccessToken
		}()
	}

	// Collect results
	var generatedTokens []string
	for i := 0; i < 10; i++ {
		select {
		case token := <-tokens:
			generatedTokens = append(generatedTokens, token)
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for token generation")
		}
	}

	assert.Len(t, generatedTokens, 10)

	// Validate all tokens concurrently
	validations := make(chan bool, 10)
	for _, token := range generatedTokens {
		go func(token string) {
			_, err := service.ValidateAccessToken(token)
			validations <- err == nil
		}(token)
	}

	// Check all validations passed
	for i := 0; i < 10; i++ {
		select {
		case valid := <-validations:
			assert.True(t, valid)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for token validation")
		}
	}
}

// Helper functions

func createTestJWTService() *JWTService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}
	return NewJWTService(cfg)
}

func createTestUser() *models.User {
	return &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}
}