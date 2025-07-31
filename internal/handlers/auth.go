package handlers

import (
	"net/http"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/database"
	"elterngeld-portal/internal/models"
	"elterngeld-portal/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db         *gorm.DB
	logger     *zap.Logger
	jwtService *auth.JWTService
	config     *config.Config
}

func NewAuthHandler(db *gorm.DB, logger *zap.Logger, jwtService *auth.JWTService, config *config.Config) *AuthHandler {
	return &AuthHandler{
		db:         db,
		logger:     logger,
		jwtService: jwtService,
		config:     config,
	}
}

// RegisterRequest represents the user registration request
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone,omitempty"`
}

// LoginRequest represents the user login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents the token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ForgotPasswordRequest represents the forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePasswordRequest represents the change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user with email verification
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid registration request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create verification token
	verificationToken := uuid.New().String()

	// Create user
	user := models.User{
		ID:                uuid.New(),
		Email:             req.Email,
		Password:          string(hashedPassword),
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Phone:             req.Phone,
		Role:              models.RoleUser,
		Status:            models.UserStatusPending,
		VerificationToken: &verificationToken,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := h.db.Create(&user).Error; err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// TODO: Send verification email
	h.logger.Info("User registered successfully", zap.String("email", user.Email), zap.String("user_id", user.ID.String()))

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully. Please check your email for verification.",
		"user_id": user.ID,
	})
}

// Login handles user authentication
// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		h.logger.Warn("Login attempt with non-existent email", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if user is verified
	if user.Status == models.UserStatusPending {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email not verified"})
		return
	}

	// Check if user is active
	if user.Status != models.UserStatusActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is not active"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.logger.Warn("Failed login attempt", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID.String(), string(user.Role))
	if err != nil {
		h.logger.Error("Failed to generate access token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		h.logger.Error("Failed to generate refresh token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	h.db.Save(&user)

	// Log successful login
	h.logger.Info("User logged in successfully", zap.String("email", user.Email), zap.String("user_id", user.ID.String()))

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	expiresAt := time.Now().Add(h.jwtService.GetAccessTokenExpiry())

	c.JSON(http.StatusOK, AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Refresh access token using refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate refresh token
	userID, err := h.jwtService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if user is active
	if user.Status != models.UserStatusActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is not active"})
		return
	}

	// Generate new tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID.String(), string(user.Role))
	if err != nil {
		h.logger.Error("Failed to generate access token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	newRefreshToken, err := h.jwtService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		h.logger.Error("Failed to generate refresh token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	expiresAt := time.Now().Add(h.jwtService.GetAccessTokenExpiry())

	c.JSON(http.StatusOK, AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	})
}

// ForgotPassword handles forgot password requests
// @Summary Request password reset
// @Description Send password reset email
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email for password reset"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Return success even if user doesn't exist for security
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
		return
	}

	// Generate reset token
	resetToken := uuid.New().String()
	resetTokenExpiry := time.Now().Add(1 * time.Hour) // 1 hour expiry

	// Update user with reset token
	user.PasswordResetToken = &resetToken
	user.PasswordResetExpiresAt = &resetTokenExpiry
	if err := h.db.Save(&user).Error; err != nil {
		h.logger.Error("Failed to save password reset token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// TODO: Send password reset email
	h.logger.Info("Password reset requested", zap.String("email", user.Email))

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
}

// ResetPassword handles password reset
// @Summary Reset password
// @Description Reset password using reset token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset password data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Find user with reset token
	var user models.User
	if err := h.db.Where("password_reset_token = ? AND password_reset_expires_at > ?", req.Token, time.Now()).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Update password and clear reset token
	user.Password = string(hashedPassword)
	user.PasswordResetToken = nil
	user.PasswordResetExpiresAt = nil
	user.UpdatedAt = time.Now()

	if err := h.db.Save(&user).Error; err != nil {
		h.logger.Error("Failed to update password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	h.logger.Info("Password reset successful", zap.String("email", user.Email))

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
}

// Logout handles user logout
// @Summary Logout user
// @Description Logout user and invalidate tokens
// @Tags authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: Implement token blacklisting
	// For now, just return success - client should discard tokens
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetMe returns current user information
// @Summary Get current user
// @Description Get current authenticated user information
// @Tags authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	c.JSON(http.StatusOK, user)
}

// UpdateMe updates current user information
// @Summary Update current user
// @Description Update current authenticated user information
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "User update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/me [put]
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Only allow certain fields to be updated
	allowedFields := []string{"first_name", "last_name", "phone", "timezone", "language"}
	filteredUpdates := make(map[string]interface{})
	for _, field := range allowedFields {
		if value, exists := updates[field]; exists {
			filteredUpdates[field] = value
		}
	}

	if len(filteredUpdates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	filteredUpdates["updated_at"] = time.Now()

	if err := h.db.Model(&user).Updates(filteredUpdates).Error; err != nil {
		h.logger.Error("Failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Fetch updated user
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated user"})
		return
	}

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	c.JSON(http.StatusOK, user)
}

// ChangePassword handles password change for authenticated users
// @Summary Change password
// @Description Change password for authenticated user
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Update password
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := h.db.Save(&user).Error; err != nil {
		h.logger.Error("Failed to update password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}

	h.logger.Info("Password changed successfully", zap.String("user_id", user.ID.String()))

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// VerifyEmail handles email verification
// @Summary Verify email
// @Description Verify user email using verification token
// @Tags authentication
// @Produce json
// @Param token query string true "Verification token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/verify-email [get]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification token is required"})
		return
	}

	// Find user with verification token
	var user models.User
	if err := h.db.Where("verification_token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification token"})
		return
	}

	// Update user status
	user.Status = models.UserStatusActive
	user.VerificationToken = nil
	user.EmailVerifiedAt = &time.Time{}
	*user.EmailVerifiedAt = time.Now()
	user.UpdatedAt = time.Now()

	if err := h.db.Save(&user).Error; err != nil {
		h.logger.Error("Failed to verify email", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify email"})
		return
	}

	h.logger.Info("Email verified successfully", zap.String("email", user.Email))

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}