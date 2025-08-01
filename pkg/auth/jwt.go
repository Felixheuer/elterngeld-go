package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// JWTService handles JWT operations
type JWTService struct {
	secretKey  []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{
		secretKey:  []byte(cfg.JWT.Secret),
		issuer:     "elterngeld-portal",
		accessTTL:  cfg.JWT.AccessExpiry,
		refreshTTL: cfg.JWT.RefreshExpiry,
	}
}

// GenerateTokenPair generates access and refresh tokens for a user
func (js *JWTService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	// Generate access token
	accessToken, err := js.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := js.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(js.accessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// GenerateAccessToken generates a JWT access token
func (js *JWTService) GenerateAccessToken(user *models.User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    js.issuer,
			Subject:   user.ID.String(),
			Audience:  []string{"elterngeld-portal-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(js.accessTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(js.secretKey)
}

// GenerateRefreshToken generates a random refresh token
func (js *JWTService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateAccessToken validates and parses an access token
func (js *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return js.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ExtractTokenFromBearer extracts token from Bearer authorization header
func ExtractTokenFromBearer(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// IsTokenExpired checks if a token is expired
func (js *JWTService) IsTokenExpired(tokenString string) bool {
	claims, err := js.ValidateAccessToken(tokenString)
	if err != nil {
		return true
	}

	return claims.RegisteredClaims.ExpiresAt.Time.Before(time.Now())
}

// GetTokenClaims extracts claims from a token without validation (for expired tokens)
func (js *JWTService) GetTokenClaims(tokenString string) (*Claims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshTokenExpiry returns the refresh token expiry time
func (js *JWTService) RefreshTokenExpiry() time.Time {
	return time.Now().Add(js.refreshTTL)
}

// AuthResponse represents authentication response
type AuthResponse struct {
	User   models.UserResponse `json:"user"`
	Tokens TokenPair           `json:"tokens"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Phone     string `json:"phone"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest represents change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

// ForgotPasswordRequest represents forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// TokenBlacklist manages blacklisted tokens (for logout)
type TokenBlacklist struct {
	tokens map[string]time.Time
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{
		tokens: make(map[string]time.Time),
	}
}

// Add adds a token to the blacklist
func (tb *TokenBlacklist) Add(tokenID string, expiresAt time.Time) {
	tb.tokens[tokenID] = expiresAt
}

// IsBlacklisted checks if a token is blacklisted
func (tb *TokenBlacklist) IsBlacklisted(tokenID string) bool {
	expiresAt, exists := tb.tokens[tokenID]
	if !exists {
		return false
	}

	// If token has expired, remove it from blacklist
	if time.Now().After(expiresAt) {
		delete(tb.tokens, tokenID)
		return false
	}

	return true
}

// Cleanup removes expired tokens from blacklist
func (tb *TokenBlacklist) Cleanup() {
	now := time.Now()
	for tokenID, expiresAt := range tb.tokens {
		if now.After(expiresAt) {
			delete(tb.tokens, tokenID)
		}
	}
}

// Global token blacklist instance
var GlobalTokenBlacklist = NewTokenBlacklist()

// ValidationError represents token validation errors
type ValidationError struct {
	Message string
	Code    string
}

func (e ValidationError) Error() string {
	return e.Message
}

// Common validation errors
var (
	ErrTokenExpired     = ValidationError{Message: "Token has expired", Code: "TOKEN_EXPIRED"}
	ErrTokenInvalid     = ValidationError{Message: "Invalid token", Code: "TOKEN_INVALID"}
	ErrTokenBlacklisted = ValidationError{Message: "Token has been revoked", Code: "TOKEN_REVOKED"}
	ErrTokenMalformed   = ValidationError{Message: "Token is malformed", Code: "TOKEN_MALFORMED"}
)

// ValidateTokenWithBlacklist validates a token and checks blacklist
func (js *JWTService) ValidateTokenWithBlacklist(tokenString string) (*Claims, error) {
	claims, err := js.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Check if token is blacklisted
	if GlobalTokenBlacklist.IsBlacklisted(claims.RegisteredClaims.ID) {
		return nil, ErrTokenBlacklisted
	}

	return claims, nil
}

// BlacklistToken adds a token to the blacklist
func (js *JWTService) BlacklistToken(tokenString string) error {
	claims, err := js.GetTokenClaims(tokenString)
	if err != nil {
		return err
	}

	GlobalTokenBlacklist.Add(claims.RegisteredClaims.ID, claims.RegisteredClaims.ExpiresAt.Time)
	return nil
}

// GetAccessTokenExpiry returns the access token expiry duration
func (js *JWTService) GetAccessTokenExpiry() time.Duration {
	return js.accessTTL
}
