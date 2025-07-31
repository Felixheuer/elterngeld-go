package handlers

import (
	"net/http"
	"strconv"
	"time"

	"elterngeld-portal/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TodoHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTodoHandler(db *gorm.DB, logger *zap.Logger) *TodoHandler {
	return &TodoHandler{
		db:     db,
		logger: logger,
	}
}

// CreateTodoRequest represents the todo creation request
type CreateTodoRequest struct {
	UserID      uuid.UUID  `json:"user_id" binding:"required"`
	LeadID      *uuid.UUID `json:"lead_id,omitempty"`
	BookingID   *uuid.UUID `json:"booking_id,omitempty"`
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority,omitempty"`
}

// UpdateTodoRequest represents the todo update request
type UpdateTodoRequest struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority,omitempty"`
	Status      string     `json:"status,omitempty"`
}

// ListTodos handles listing todos with filtering
// @Summary List todos
// @Description Get list of todos with filtering options
// @Tags todos
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Param assigned_to query string false "Filter by assigned user"
// @Param my_todos query bool false "Show only my todos"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/todos [get]
func (h *TodoHandler) ListTodos(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("user_role")

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Parse filters
	status := c.Query("status")
	assignedTo := c.Query("assigned_to")
	myTodos := c.Query("my_todos") == "true"

	// Build query
	query := h.db.Model(&models.Todo{})

	// Role-based filtering
	if userRole == "user" {
		// Users can only see todos assigned to them
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" || userRole == "berater" {
		if myTodos {
			// Show todos created by this berater
			query = query.Where("assigned_by_id = ?", userID)
		} else {
			// Show todos assigned to users they can access
			query = query.Joins("LEFT JOIN leads ON todos.lead_id = leads.id").
				Where("todos.assigned_by_id = ? OR leads.assigned_to_id = ?", userID, userID)
		}
	}
	// Admins can see all todos

	// Apply filters
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if assignedTo != "" {
		query = query.Where("user_id = ?", assignedTo)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get todos with preloaded relations
	var todos []models.Todo
	if err := query.Preload("User").Preload("AssignedBy").Preload("Lead").Preload("Booking").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&todos).Error; err != nil {
		h.logger.Error("Failed to fetch todos", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"todos": todos,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// CreateTodo handles creating a new todo
// @Summary Create todo
// @Description Create a new todo for a user (Berater/Admin only)
// @Tags todos
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateTodoRequest true "Todo data"
// @Success 201 {object} models.Todo
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /api/v1/todos [post]
func (h *TodoHandler) CreateTodo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("user_role")

	// Only beraters and admins can create todos for other users
	if userRole != "berater" && userRole != "junior_berater" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var req CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify target user exists
	var targetUser models.User
	if err := h.db.Where("id = ?", req.UserID).First(&targetUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target user not found"})
		} else {
			h.logger.Error("Failed to fetch target user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify target user"})
		}
		return
	}

	// Verify lead/booking exists if provided
	if req.LeadID != nil {
		var lead models.Lead
		if err := h.db.Where("id = ?", *req.LeadID).First(&lead).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lead ID"})
			return
		}
	}

	if req.BookingID != nil {
		var booking models.Booking
		if err := h.db.Where("id = ?", *req.BookingID).First(&booking).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}
	}

	// Set default priority if not provided
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	// Create todo
	todo := models.Todo{
		ID:           uuid.New(),
		UserID:       req.UserID,
		AssignedByID: userID.(uuid.UUID),
		LeadID:       req.LeadID,
		BookingID:    req.BookingID,
		Title:        req.Title,
		Description:  req.Description,
		Status:       models.TodoStatusOpen,
		Priority:     priority,
		DueDate:      req.DueDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.db.Create(&todo).Error; err != nil {
		h.logger.Error("Failed to create todo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create todo"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      req.LeadID,
		Type:        models.ActivityTypeTodoCreated,
		Description: "Todo created: " + todo.Title,
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	h.logger.Info("Todo created successfully", 
		zap.String("todo_id", todo.ID.String()),
		zap.String("assigned_by", userID.(uuid.UUID).String()),
		zap.String("assigned_to", req.UserID.String()))

	// TODO: Send email notification to user
	h.logger.Info("Todo email notification should be sent", 
		zap.String("recipient", targetUser.Email),
		zap.String("todo_title", todo.Title))

	// Load relations for response
	h.db.Preload("User").Preload("AssignedBy").Preload("Lead").Preload("Booking").First(&todo, todo.ID)

	c.JSON(http.StatusCreated, todo)
}

// GetTodo handles getting a specific todo
// @Summary Get todo by ID
// @Description Get todo details
// @Tags todos
// @Security BearerAuth
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} models.Todo
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/todos/{id} [get]
func (h *TodoHandler) GetTodo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	todoID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var todo models.Todo
	query := h.db.Where("id = ?", todoID)

	// Role-based access control
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" || userRole == "berater" {
		query = query.Where("assigned_by_id = ? OR user_id = ?", userID, userID)
	}
	// Admins can see all todos

	if err := query.Preload("User").Preload("AssignedBy").Preload("Lead").Preload("Booking").First(&todo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		} else {
			h.logger.Error("Failed to fetch todo", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo"})
		}
		return
	}

	c.JSON(http.StatusOK, todo)
}

// UpdateTodo handles updating a todo
// @Summary Update todo
// @Description Update todo information
// @Tags todos
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Param request body UpdateTodoRequest true "Todo update data"
// @Success 200 {object} models.Todo
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/todos/{id} [put]
func (h *TodoHandler) UpdateTodo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	todoID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var todo models.Todo
	query := h.db.Where("id = ?", todoID)

	// Role-based access control
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" || userRole == "berater" {
		query = query.Where("assigned_by_id = ?", userID)
	}

	if err := query.First(&todo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		} else {
			h.logger.Error("Failed to fetch todo", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo"})
		}
		return
	}

	var req UpdateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Build updates
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Priority != "" {
		updates["priority"] = req.Priority
	}
	if req.Status != "" {
		updates["status"] = req.Status
		// Set completion timestamp if marking as completed
		if req.Status == string(models.TodoStatusCompleted) && todo.CompletedAt == nil {
			now := time.Now()
			updates["completed_at"] = &now
		}
	}
	if req.DueDate != nil {
		updates["due_date"] = req.DueDate
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	updates["updated_at"] = time.Now()

	if err := h.db.Model(&todo).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update todo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update todo"})
		return
	}

	// Create activity log if status changed
	if req.Status != "" {
		activity := models.Activity{
			ID:          uuid.New(),
			UserID:      userID.(uuid.UUID),
			LeadID:      todo.LeadID,
			Type:        models.ActivityTypeTodoUpdated,
			Description: "Todo status changed to: " + req.Status,
			CreatedAt:   time.Now(),
		}
		h.db.Create(&activity)
	}

	// Fetch updated todo
	if err := h.db.Preload("User").Preload("AssignedBy").Preload("Lead").Preload("Booking").First(&todo, "id = ?", todoID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated todo"})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// CompleteTodo handles marking a todo as completed
// @Summary Complete todo
// @Description Mark todo as completed
// @Tags todos
// @Security BearerAuth
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} models.Todo
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/todos/{id}/complete [patch]
func (h *TodoHandler) CompleteTodo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	todoID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var todo models.Todo
	query := h.db.Where("id = ?", todoID)

	// Users can complete their own todos, beraters can complete any todo they created
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" || userRole == "berater" {
		query = query.Where("assigned_by_id = ? OR user_id = ?", userID, userID)
	}

	if err := query.First(&todo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		} else {
			h.logger.Error("Failed to fetch todo", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo"})
		}
		return
	}

	// Mark as completed
	now := time.Now()
	todo.Status = models.TodoStatusCompleted
	todo.CompletedAt = &now
	todo.UpdatedAt = now

	if err := h.db.Save(&todo).Error; err != nil {
		h.logger.Error("Failed to complete todo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete todo"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      todo.LeadID,
		Type:        models.ActivityTypeTodoCompleted,
		Description: "Todo completed: " + todo.Title,
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	h.logger.Info("Todo completed", zap.String("todo_id", todoID))

	// Load relations for response
	h.db.Preload("User").Preload("AssignedBy").Preload("Lead").Preload("Booking").First(&todo, todo.ID)

	c.JSON(http.StatusOK, todo)
}

// DeleteTodo handles deleting a todo
// @Summary Delete todo
// @Description Delete a todo (Berater/Admin only)
// @Tags todos
// @Security BearerAuth
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/todos/{id} [delete]
func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	todoID := c.Param("id")
	userRole, _ := c.Get("user_role")

	// Only beraters and admins can delete todos
	if userRole != "berater" && userRole != "junior_berater" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var todo models.Todo
	query := h.db.Where("id = ?", todoID)

	// Beraters can only delete todos they created
	if userRole != "admin" {
		query = query.Where("assigned_by_id = ?", userID)
	}

	if err := query.First(&todo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		} else {
			h.logger.Error("Failed to fetch todo", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo"})
		}
		return
	}

	// Soft delete
	if err := h.db.Delete(&todo).Error; err != nil {
		h.logger.Error("Failed to delete todo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete todo"})
		return
	}

	h.logger.Info("Todo deleted successfully", zap.String("todo_id", todoID))

	c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
}