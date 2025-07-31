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

type LeadHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewLeadHandler(db *gorm.DB, logger *zap.Logger) *LeadHandler {
	return &LeadHandler{
		db:     db,
		logger: logger,
	}
}

// CreateLeadRequest represents the lead creation request
type CreateLeadRequest struct {
	Source        models.LeadSource   `json:"source" binding:"required"`
	Title         string              `json:"title" binding:"required"`
	Description   string              `json:"description,omitempty"`
	Priority      models.LeadPriority `json:"priority,omitempty"`
	EstimatedValue *float64           `json:"estimated_value,omitempty"`
	CompanyName   string              `json:"company_name,omitempty"`
	ContactEmail  string              `json:"contact_email,omitempty"`
	ContactPhone  string              `json:"contact_phone,omitempty"`
	UTMSource     string              `json:"utm_source,omitempty"`
	UTMCampaign   string              `json:"utm_campaign,omitempty"`
	UTMMedium     string              `json:"utm_medium,omitempty"`
	Notes         string              `json:"notes,omitempty"`
}

// UpdateLeadRequest represents the lead update request
type UpdateLeadRequest struct {
	Title          string              `json:"title,omitempty"`
	Description    string              `json:"description,omitempty"`
	Priority       models.LeadPriority `json:"priority,omitempty"`
	EstimatedValue *float64            `json:"estimated_value,omitempty"`
	CompanyName    string              `json:"company_name,omitempty"`
	ContactEmail   string              `json:"contact_email,omitempty"`
	ContactPhone   string              `json:"contact_phone,omitempty"`
	Notes          string              `json:"notes,omitempty"`
	FollowUpDate   *time.Time          `json:"follow_up_date,omitempty"`
}

// UpdateLeadStatusRequest represents the lead status update request
type UpdateLeadStatusRequest struct {
	Status models.LeadStatus `json:"status" binding:"required"`
	Notes  string            `json:"notes,omitempty"`
}

// AssignLeadRequest represents the lead assignment request
type AssignLeadRequest struct {
	AssignedToID uuid.UUID `json:"assigned_to_id" binding:"required"`
	Notes        string    `json:"notes,omitempty"`
}

// CreateCommentRequest represents the comment creation request
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

// LeadResponse represents a lead with related data
type LeadResponse struct {
	*models.Lead
	User         *models.User           `json:"user,omitempty"`
	AssignedTo   *models.User           `json:"assigned_to,omitempty"`
	Booking      *models.Booking        `json:"booking,omitempty"`
	Activities   []models.Activity      `json:"activities,omitempty"`
	Comments     []models.Comment       `json:"comments,omitempty"`
	Todos        []models.Todo          `json:"todos,omitempty"`
	Documents    []models.Document      `json:"documents,omitempty"`
}

// ListLeads handles listing leads with filtering and pagination
// @Summary List leads
// @Description Get list of leads with filtering options
// @Tags leads
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Param priority query string false "Filter by priority"
// @Param source query string false "Filter by source"
// @Param assigned_to query string false "Filter by assigned user"
// @Param search query string false "Search in title or description"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/leads [get]
func (h *LeadHandler) ListLeads(c *gin.Context) {
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
	priority := c.Query("priority")
	source := c.Query("source")
	assignedTo := c.Query("assigned_to")
	search := c.Query("search")
	myLeads := c.Query("my_leads") == "true"

	// Build query
	query := h.db.Model(&models.Lead{})

	// Role-based filtering
	if userRole == "user" {
		// Users can only see their own leads
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		// Junior beraters can see assigned leads or unassigned ones
		if myLeads {
			query = query.Where("assigned_to_id = ?", userID)
		} else {
			query = query.Where("assigned_to_id = ? OR assigned_to_id IS NULL", userID)
		}
	} else if userRole == "berater" {
		// Beraters can see all leads but have option to filter their own
		if myLeads {
			query = query.Where("assigned_to_id = ?", userID)
		}
	}
	// Admins can see all leads without restrictions

	// Apply filters
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if assignedTo != "" {
		query = query.Where("assigned_to_id = ?", assignedTo)
	}
	if search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get leads with preloaded relations
	var leads []models.Lead
	if err := query.Preload("User").Preload("AssignedTo").Preload("Booking").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&leads).Error; err != nil {
		h.logger.Error("Failed to fetch leads", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leads"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"leads": leads,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// CreateLead handles creating a new lead
// @Summary Create lead
// @Description Create a new lead
// @Tags leads
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateLeadRequest true "Lead data"
// @Success 201 {object} LeadResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/leads [post]
func (h *LeadHandler) CreateLead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Set default priority if not provided
	priority := req.Priority
	if priority == "" {
		priority = models.LeadPriorityMedium
	}

	// Create lead
	lead := models.Lead{
		ID:             uuid.New(),
		UserID:         &userID.(uuid.UUID),
		Source:         req.Source,
		Status:         models.LeadStatusNew,
		Priority:       priority,
		Title:          req.Title,
		Description:    req.Description,
		EstimatedValue: req.EstimatedValue,
		CompanyName:    req.CompanyName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		UTMSource:      req.UTMSource,
		UTMCampaign:    req.UTMCampaign,
		UTMMedium:      req.UTMMedium,
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.db.Create(&lead).Error; err != nil {
		h.logger.Error("Failed to create lead", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lead"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:           uuid.New(),
		UserID:       userID.(uuid.UUID),
		LeadID:       &lead.ID,
		Type:         models.ActivityTypeLeadCreated,
		Description:  "Lead created: " + lead.Title,
		CreatedAt:    time.Now(),
	}
	h.db.Create(&activity)

	h.logger.Info("Lead created successfully", 
		zap.String("lead_id", lead.ID.String()),
		zap.String("user_id", userID.(uuid.UUID).String()))

	// Prepare response
	response := &LeadResponse{
		Lead: &lead,
	}

	c.JSON(http.StatusCreated, response)
}

// GetLead handles getting a specific lead
// @Summary Get lead by ID
// @Description Get lead details with all related data
// @Tags leads
// @Security BearerAuth
// @Produce json
// @Param id path string true "Lead ID"
// @Success 200 {object} LeadResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id} [get]
func (h *LeadHandler) GetLead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	leadID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var lead models.Lead
	query := h.db.Where("id = ?", leadID)

	// Role-based access control
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		query = query.Where("assigned_to_id = ? OR assigned_to_id IS NULL", userID)
	}
	// Beraters and Admins can see all leads

	if err := query.Preload("User").Preload("AssignedTo").Preload("Booking").
		Preload("Activities").Preload("Documents").First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			h.logger.Error("Failed to fetch lead", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	// Get comments
	var comments []models.Comment
	h.db.Where("lead_id = ?", lead.ID).Preload("User").Order("created_at ASC").Find(&comments)

	// Get todos
	var todos []models.Todo
	h.db.Where("lead_id = ?", lead.ID).Preload("AssignedTo").Order("created_at DESC").Find(&todos)

	response := &LeadResponse{
		Lead:       &lead,
		User:       lead.User,
		AssignedTo: lead.AssignedTo,
		Booking:    lead.Booking,
		Activities: lead.Activities,
		Comments:   comments,
		Todos:      todos,
		Documents:  lead.Documents,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateLead handles updating a lead
// @Summary Update lead
// @Description Update lead information
// @Tags leads
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Lead ID"
// @Param request body UpdateLeadRequest true "Lead update data"
// @Success 200 {object} models.Lead
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id} [put]
func (h *LeadHandler) UpdateLead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	leadID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var lead models.Lead
	query := h.db.Where("id = ?", leadID)

	// Role-based access control
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		query = query.Where("assigned_to_id = ?", userID)
	}

	if err := query.First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			h.logger.Error("Failed to fetch lead", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	var req UpdateLeadRequest
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
	if req.EstimatedValue != nil {
		updates["estimated_value"] = req.EstimatedValue
	}
	if req.CompanyName != "" {
		updates["company_name"] = req.CompanyName
	}
	if req.ContactEmail != "" {
		updates["contact_email"] = req.ContactEmail
	}
	if req.ContactPhone != "" {
		updates["contact_phone"] = req.ContactPhone
	}
	if req.Notes != "" {
		updates["notes"] = req.Notes
	}
	if req.FollowUpDate != nil {
		updates["follow_up_date"] = req.FollowUpDate
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	updates["updated_at"] = time.Now()

	if err := h.db.Model(&lead).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update lead", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lead"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeLeadUpdated,
		Description: "Lead updated",
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	// Fetch updated lead
	if err := h.db.First(&lead, "id = ?", leadID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated lead"})
		return
	}

	c.JSON(http.StatusOK, lead)
}

// DeleteLead handles deleting a lead
// @Summary Delete lead
// @Description Delete a lead (soft delete)
// @Tags leads
// @Security BearerAuth
// @Produce json
// @Param id path string true "Lead ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id} [delete]
func (h *LeadHandler) DeleteLead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	leadID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var lead models.Lead
	query := h.db.Where("id = ?", leadID)

	// Only beraters and admins can delete leads
	if userRole != "berater" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	if err := query.First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			h.logger.Error("Failed to fetch lead", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	// Soft delete
	if err := h.db.Delete(&lead).Error; err != nil {
		h.logger.Error("Failed to delete lead", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete lead"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeLeadDeleted,
		Description: "Lead deleted",
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	h.logger.Info("Lead deleted successfully", zap.String("lead_id", leadID))

	c.JSON(http.StatusOK, gin.H{"message": "Lead deleted successfully"})
}

// UpdateLeadStatus handles updating lead status
// @Summary Update lead status
// @Description Update lead status with workflow validation
// @Tags leads
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Lead ID"
// @Param request body UpdateLeadStatusRequest true "Status update data"
// @Success 200 {object} models.Lead
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id}/status [patch]
func (h *LeadHandler) UpdateLeadStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	leadID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var lead models.Lead
	query := h.db.Where("id = ?", leadID)

	// Role-based access control
	if userRole == "user" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Users cannot update lead status"})
		return
	} else if userRole == "junior_berater" {
		query = query.Where("assigned_to_id = ?", userID)
	}

	if err := query.First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			h.logger.Error("Failed to fetch lead", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	var req UpdateLeadStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// TODO: Implement status transition validation
	// For now, allow any status change

	oldStatus := lead.Status
	lead.Status = req.Status
	lead.UpdatedAt = time.Now()

	// Set qualified/disqualified timestamps
	if req.Status == models.LeadStatusQualified && lead.QualifiedAt == nil {
		now := time.Now()
		lead.QualifiedAt = &now
	} else if req.Status == models.LeadStatusUnqualified && lead.DisqualifiedAt == nil {
		now := time.Now()
		lead.DisqualifiedAt = &now
	}

	if err := h.db.Save(&lead).Error; err != nil {
		h.logger.Error("Failed to update lead status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lead status"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeStatusChanged,
		Description: "Status changed from " + string(oldStatus) + " to " + string(req.Status),
		Metadata:    req.Notes,
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	h.logger.Info("Lead status updated", 
		zap.String("lead_id", leadID),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(req.Status)))

	c.JSON(http.StatusOK, lead)
}

// AssignLead handles assigning a lead to a berater
// @Summary Assign lead
// @Description Assign lead to a berater (Berater/Admin only)
// @Tags leads
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Lead ID"
// @Param request body AssignLeadRequest true "Assignment data"
// @Success 200 {object} models.Lead
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id}/assign [post]
func (h *LeadHandler) AssignLead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	leadID := c.Param("id")

	var lead models.Lead
	if err := h.db.Where("id = ?", leadID).First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			h.logger.Error("Failed to fetch lead", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	var req AssignLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Verify assigned user exists and has appropriate role
	var assignedUser models.User
	if err := h.db.Where("id = ? AND role IN ?", req.AssignedToID, 
		[]string{"berater", "junior_berater"}).First(&assignedUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user or user cannot be assigned leads"})
		} else {
			h.logger.Error("Failed to fetch assigned user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assigned user"})
		}
		return
	}

	oldAssignedTo := lead.AssignedToID
	lead.AssignedToID = &req.AssignedToID
	lead.AssignedAt = &time.Time{}
	*lead.AssignedAt = time.Now()
	lead.UpdatedAt = time.Now()

	if err := h.db.Save(&lead).Error; err != nil {
		h.logger.Error("Failed to assign lead", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign lead"})
		return
	}

	// Create activity log
	description := "Lead assigned to " + assignedUser.FirstName + " " + assignedUser.LastName
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeLeadAssigned,
		Description: description,
		Metadata:    req.Notes,
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	h.logger.Info("Lead assigned", 
		zap.String("lead_id", leadID),
		zap.String("assigned_to", req.AssignedToID.String()))

	// TODO: Send assignment notification email

	c.JSON(http.StatusOK, lead)
}

// ListLeadComments handles listing comments for a lead
// @Summary List lead comments
// @Description Get comments for a specific lead
// @Tags leads
// @Security BearerAuth
// @Produce json
// @Param id path string true "Lead ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id}/comments [get]
func (h *LeadHandler) ListLeadComments(c *gin.Context) {
	leadID := c.Param("id")

	// Verify lead exists and user has access
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("user_role")

	var lead models.Lead
	query := h.db.Where("id = ?", leadID)

	// Role-based access control
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		query = query.Where("assigned_to_id = ? OR assigned_to_id IS NULL", userID)
	}

	if err := query.First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	// Get comments
	var comments []models.Comment
	if err := h.db.Where("lead_id = ?", leadID).Preload("User").
		Order("created_at ASC").Find(&comments).Error; err != nil {
		h.logger.Error("Failed to fetch comments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
	})
}

// CreateLeadComment handles creating a comment for a lead
// @Summary Create lead comment
// @Description Add a comment to a lead
// @Tags leads
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Lead ID"
// @Param request body CreateCommentRequest true "Comment data"
// @Success 201 {object} models.Comment
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/leads/{id}/comments [post]
func (h *LeadHandler) CreateLeadComment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	leadID := c.Param("id")

	// Verify lead exists and user has access
	userRole, _ := c.Get("user_role")

	var lead models.Lead
	query := h.db.Where("id = ?", leadID)

	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		query = query.Where("assigned_to_id = ? OR assigned_to_id IS NULL", userID)
	}

	if err := query.First(&lead).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lead not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lead"})
		}
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Create comment
	comment := models.Comment{
		ID:        uuid.New(),
		UserID:    userID.(uuid.UUID),
		LeadID:    &lead.ID,
		Content:   req.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.Create(&comment).Error; err != nil {
		h.logger.Error("Failed to create comment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	// Create activity log
	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeCommentAdded,
		Description: "Comment added",
		CreatedAt:   time.Now(),
	}
	h.db.Create(&activity)

	// Load user relation
	h.db.Preload("User").First(&comment, comment.ID)

	c.JSON(http.StatusCreated, comment)
}