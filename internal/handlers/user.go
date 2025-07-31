package handlers

import (
	"net/http"
	"strconv"
	"time"

	"elterngeld-portal/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserHandler(db *gorm.DB, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		db:     db,
		logger: logger,
	}
}

// UpdateUserRequest represents the user update request
type UpdateUserRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	Language  string `json:"language,omitempty"`
}

// CreateUserRequest represents the admin create user request
type CreateUserRequest struct {
	Email     string           `json:"email" binding:"required,email"`
	Password  string           `json:"password" binding:"required,min=8"`
	FirstName string           `json:"first_name" binding:"required"`
	LastName  string           `json:"last_name" binding:"required"`
	Phone     string           `json:"phone,omitempty"`
	Role      models.UserRole  `json:"role" binding:"required"`
	Status    models.UserStatus `json:"status,omitempty"`
}

// ListUsers handles listing users with pagination and filtering
// @Summary List users
// @Description Get list of users (Berater/Admin only)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param role query string false "Filter by role"
// @Param status query string false "Filter by status"
// @Param search query string false "Search in name or email"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Parse filters
	role := c.Query("role")
	status := c.Query("status")
	search := c.Query("search")

	// Build query
	query := h.db.Model(&models.User{})

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if search != "" {
		query = query.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get users
	var users []models.User
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		h.logger.Error("Failed to fetch users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Remove sensitive data
	for i := range users {
		users[i].Password = ""
		users[i].VerificationToken = nil
		users[i].PasswordResetToken = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetUser handles getting a specific user
// @Summary Get user by ID
// @Description Get user information by ID (with ownership/role checks)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			h.logger.Error("Failed to fetch user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		}
		return
	}

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles updating a user
// @Summary Update user
// @Description Update user information (ownership/admin required)
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserRequest true "User update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			h.logger.Error("Failed to fetch user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		}
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	if req.Language != "" {
		updates["language"] = req.Language
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	updates["updated_at"] = time.Now()

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
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

// DeleteUser handles deleting a user (Admin only)
// @Summary Delete user
// @Description Delete user (Admin only)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			h.logger.Error("Failed to fetch user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		}
		return
	}

	// Soft delete the user
	if err := h.db.Delete(&user).Error; err != nil {
		h.logger.Error("Failed to delete user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	h.logger.Info("User deleted successfully", zap.String("user_id", userID))

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// AdminCreateUser handles creating a new user (Admin only)
// @Summary Create user (Admin)
// @Description Create a new user (Admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User creation data"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/v1/admin/users [post]
func (h *UserHandler) AdminCreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid create user request", zap.Error(err))
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

	// Set default status if not provided
	status := req.Status
	if status == "" {
		status = models.UserStatusActive // Admin created users are active by default
	}

	// Create user
	user := models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      req.Role,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// If active, mark email as verified
	if status == models.UserStatusActive {
		now := time.Now()
		user.EmailVerifiedAt = &now
	}

	if err := h.db.Create(&user).Error; err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	h.logger.Info("User created by admin", zap.String("email", user.Email), zap.String("user_id", user.ID.String()))

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	c.JSON(http.StatusCreated, user)
}

// AdminChangeUserRole handles changing user role (Admin only)
// @Summary Change user role (Admin)
// @Description Change user role (Admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body map[string]interface{} true "Role change data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/role [put]
func (h *UserHandler) AdminChangeUserRole(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			h.logger.Error("Failed to fetch user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		}
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	newRole, exists := req["role"]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role is required"})
		return
	}

	// Validate role
	roleStr, ok := newRole.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role format"})
		return
	}

	// Check if role is valid
	validRoles := []string{"user", "junior_berater", "berater", "admin"}
	isValid := false
	for _, validRole := range validRoles {
		if roleStr == validRole {
			isValid = true
			break
		}
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	oldRole := user.Role
	user.Role = models.UserRole(roleStr)
	user.UpdatedAt = time.Now()

	if err := h.db.Save(&user).Error; err != nil {
		h.logger.Error("Failed to update user role", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	h.logger.Info("User role changed", 
		zap.String("user_id", userID), 
		zap.String("old_role", string(oldRole)),
		zap.String("new_role", string(user.Role)))

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	c.JSON(http.StatusOK, user)
}

// AdminChangeUserStatus handles changing user status (Admin only)
// @Summary Change user status (Admin)
// @Description Change user status (Admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body map[string]interface{} true "Status change data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/status [put]
func (h *UserHandler) AdminChangeUserStatus(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			h.logger.Error("Failed to fetch user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		}
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	newStatus, exists := req["status"]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}

	// Validate status
	statusStr, ok := newStatus.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status format"})
		return
	}

	// Check if status is valid
	validStatuses := []string{"pending", "active", "suspended", "inactive"}
	isValid := false
	for _, validStatus := range validStatuses {
		if statusStr == validStatus {
			isValid = true
			break
		}
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	oldStatus := user.Status
	user.Status = models.UserStatus(statusStr)
	user.UpdatedAt = time.Now()

	// If changing to active and email not verified, mark as verified
	if user.Status == models.UserStatusActive && user.EmailVerifiedAt == nil {
		now := time.Now()
		user.EmailVerifiedAt = &now
	}

	if err := h.db.Save(&user).Error; err != nil {
		h.logger.Error("Failed to update user status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	h.logger.Info("User status changed", 
		zap.String("user_id", userID), 
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(user.Status)))

	// Remove sensitive data
	user.Password = ""
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	c.JSON(http.StatusOK, user)
}