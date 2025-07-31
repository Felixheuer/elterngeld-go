package handlers

import (
	"net/http"
	"time"

	"elterngeld-portal/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContactHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewContactHandler(db *gorm.DB, logger *zap.Logger) *ContactHandler {
	return &ContactHandler{
		db:     db,
		logger: logger,
	}
}

// ContactFormRequest represents the contact form submission
type ContactFormRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone,omitempty"`
	Company      string `json:"company,omitempty"`
	Subject      string `json:"subject" binding:"required"`
	Message      string `json:"message" binding:"required"`
	PreferredDate *time.Time `json:"preferred_date,omitempty"`
	
	// UTM tracking parameters
	UTMSource    string `json:"utm_source,omitempty"`
	UTMCampaign  string `json:"utm_campaign,omitempty"`
	UTMMedium    string `json:"utm_medium,omitempty"`
	UTMTerm      string `json:"utm_term,omitempty"`
	UTMContent   string `json:"utm_content,omitempty"`
	
	// Additional tracking
	PageURL      string `json:"page_url,omitempty"`
	Referrer     string `json:"referrer,omitempty"`
}

// PreTalkBookingRequest represents a free 15-min consultation booking
type PreTalkBookingRequest struct {
	Name         string     `json:"name" binding:"required"`
	Email        string     `json:"email" binding:"required,email"`
	Phone        string     `json:"phone,omitempty"`
	TimeslotID   uuid.UUID  `json:"timeslot_id" binding:"required"`
	Message      string     `json:"message,omitempty"`
	
	// UTM tracking
	UTMSource    string `json:"utm_source,omitempty"`
	UTMCampaign  string `json:"utm_campaign,omitempty"`
	UTMMedium    string `json:"utm_medium,omitempty"`
}

// SubmitContactForm handles contact form submissions
// @Summary Submit contact form
// @Description Submit a contact form (creates a lead automatically)
// @Tags contact
// @Accept json
// @Produce json
// @Param request body ContactFormRequest true "Contact form data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/contact [post]
func (h *ContactHandler) SubmitContactForm(c *gin.Context) {
	var req ContactFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid contact form request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Check if user is authenticated
	var userID *uuid.UUID
	if rawUserID, exists := c.Get("user_id"); exists {
		if uid, ok := rawUserID.(uuid.UUID); ok {
			userID = &uid
		}
	}

	// Check if user already exists by email
	var existingUser models.User
	userExists := h.db.Where("email = ?", req.Email).First(&existingUser).Error == nil

	// If user doesn't exist and we have an authenticated user, that's inconsistent
	if userID != nil && !userExists {
		// This shouldn't happen, but handle gracefully
		h.logger.Warn("Authenticated user not found in database", zap.String("user_id", userID.String()))
		userID = nil
	}

	// If user exists but we're not authenticated, use existing user
	if userExists && userID == nil {
		userID = &existingUser.ID
	}

	// Start database transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create contact form record
	contactForm := models.ContactForm{
		ID:               uuid.New(),
		UserID:           userID,
		Name:             req.Name,
		Email:            req.Email,
		Phone:            req.Phone,
		Company:          req.Company,
		Subject:          req.Subject,
		Message:          req.Message,
		PreferredDate:    req.PreferredDate,
		UTMSource:        req.UTMSource,
		UTMCampaign:      req.UTMCampaign,
		UTMMedium:        req.UTMMedium,
		UTMTerm:          req.UTMTerm,
		UTMContent:       req.UTMContent,
		PageURL:          req.PageURL,
		Referrer:         req.Referrer,
		Status:           models.ContactFormStatusNew,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := tx.Create(&contactForm).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to create contact form", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save contact form"})
		return
	}

	// Create associated lead
	leadTitle := "Contact Form: " + req.Subject
	leadDescription := "Contact form submission from " + req.Name + "\n\n" + req.Message

	// Determine lead source
	var leadSource models.LeadSource = models.LeadSourceWebsite
	if req.UTMSource != "" {
		switch req.UTMSource {
		case "google":
			leadSource = models.LeadSourceGoogle
		case "facebook":
			leadSource = models.LeadSourceSocial
		case "email":
			leadSource = models.LeadSourceEmail
		default:
			leadSource = models.LeadSourceWebsite
		}
	}

	lead := models.Lead{
		ID:               uuid.New(),
		UserID:           userID,
		ContactFormID:    &contactForm.ID,
		Source:           leadSource,
		Status:           models.LeadStatusNew,
		Priority:         models.LeadPriorityMedium,
		Title:            leadTitle,
		Description:      leadDescription,
		CompanyName:      req.Company,
		ContactEmail:     req.Email,
		ContactPhone:     req.Phone,
		UTMSource:        req.UTMSource,
		UTMCampaign:      req.UTMCampaign,
		UTMMedium:        req.UTMMedium,
		UTMTerm:          req.UTMTerm,
		UTMContent:       req.UTMContent,
		FollowUpDate:     req.PreferredDate,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := tx.Create(&lead).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to create lead from contact form", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process contact form"})
		return
	}

	// Update contact form with lead ID
	contactForm.LeadID = &lead.ID
	if err := tx.Save(&contactForm).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to update contact form with lead ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process contact form"})
		return
	}

	// Create activity log
	var activityUserID uuid.UUID
	if userID != nil {
		activityUserID = *userID
	} else {
		// Create a system activity for anonymous submissions
		activityUserID = uuid.New() // This should be a system user ID in production
	}

	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      activityUserID,
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeLeadCreated,
		Description: "Lead created from contact form submission",
		Metadata:    "Contact form ID: " + contactForm.ID.String(),
		CreatedAt:   time.Now(),
	}
	tx.Create(&activity)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("Failed to commit contact form transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process contact form"})
		return
	}

	h.logger.Info("Contact form submitted successfully", 
		zap.String("contact_form_id", contactForm.ID.String()),
		zap.String("lead_id", lead.ID.String()),
		zap.String("email", req.Email))

	// TODO: Send confirmation email to user
	// TODO: Send notification email to beraters

	c.JSON(http.StatusCreated, gin.H{
		"message":          "Contact form submitted successfully",
		"contact_form_id":  contactForm.ID,
		"lead_id":          lead.ID,
		"reference_number": "CF-" + contactForm.ID.String()[:8],
	})
}

// BookPreTalk handles free 15-minute consultation booking
// @Summary Book free consultation
// @Description Book a free 15-minute consultation (for specific packages)
// @Tags contact
// @Accept json
// @Produce json
// @Param request body PreTalkBookingRequest true "Pre-talk booking data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/contact/pre-talk [post]
func (h *ContactHandler) BookPreTalk(c *gin.Context) {
	var req PreTalkBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid pre-talk booking request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Check if user is authenticated
	var userID *uuid.UUID
	if rawUserID, exists := c.Get("user_id"); exists {
		if uid, ok := rawUserID.(uuid.UUID); ok {
			userID = &uid
		}
	}

	// Verify timeslot exists and is available
	var timeslot models.Timeslot
	if err := h.db.Where("id = ? AND is_available = ?", req.TimeslotID, true).First(&timeslot).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Timeslot not found or not available"})
		} else {
			h.logger.Error("Failed to fetch timeslot", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify timeslot"})
		}
		return
	}

	// Check if timeslot is still available (not overbooked)
	var bookingCount int64
	h.db.Model(&models.Booking{}).Where("timeslot_id = ? AND status NOT IN (?)", 
		timeslot.ID, []string{"cancelled", "completed"}).Count(&bookingCount)
	
	if bookingCount >= int64(timeslot.MaxBookings) {
		c.JSON(http.StatusConflict, gin.H{"error": "Timeslot is no longer available"})
		return
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find free consultation package
	var preTalkPackage models.Package
	if err := tx.Where("type = ? AND name ILIKE ? AND price = ?", 
		models.PackageTypeService, "%vorgespräch%", 0.0).First(&preTalkPackage).Error; err != nil {
		// If no free package exists, create a placeholder
		h.logger.Warn("No free consultation package found, using placeholder")
	}

	// Generate booking reference
	bookingRef := "PT" + time.Now().Format("20060102") + "-" + uuid.New().String()[:8]

	// Create booking for pre-talk
	booking := models.Booking{
		ID:               uuid.New(),
		UserID:           userID, // Can be nil for anonymous bookings
		TimeslotID:       &req.TimeslotID,
		BookingReference: bookingRef,
		Status:           models.BookingStatusPending,
		TotalPrice:       0.0, // Free consultation
		Currency:         "EUR",
		BookingDate:      time.Now(),
		ContactFirstName: req.Name,
		ContactEmail:     req.Email,
		ContactPhone:     req.Phone,
		ContactCompleted: true,
		Notes:            req.Message,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if preTalkPackage.ID != uuid.Nil {
		booking.PackageID = preTalkPackage.ID
	}

	if err := tx.Create(&booking).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to create pre-talk booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Create associated lead
	leadTitle := "Free Consultation: " + req.Name
	leadDescription := "Free 15-minute consultation booking"
	if req.Message != "" {
		leadDescription += "\n\nMessage: " + req.Message
	}

	lead := models.Lead{
		ID:          uuid.New(),
		UserID:      userID,
		BookingID:   &booking.ID,
		Source:      models.LeadSourceWebsite,
		Status:      models.LeadStatusNew,
		Priority:    models.LeadPriorityHigh, // Pre-talks are high priority
		Title:       leadTitle,
		Description: leadDescription,
		ContactEmail: req.Email,
		ContactPhone: req.Phone,
		UTMSource:   req.UTMSource,
		UTMCampaign: req.UTMCampaign,
		UTMMedium:   req.UTMMedium,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := tx.Create(&lead).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to create lead from pre-talk booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process booking"})
		return
	}

	// Update booking with lead ID
	booking.LeadID = &lead.ID
	if err := tx.Save(&booking).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to update booking with lead ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process booking"})
		return
	}

	// Create activity log
	var activityUserID uuid.UUID
	if userID != nil {
		activityUserID = *userID
	} else {
		activityUserID = uuid.New() // System user for anonymous bookings
	}

	activity := models.Activity{
		ID:          uuid.New(),
		UserID:      activityUserID,
		LeadID:      &lead.ID,
		Type:        models.ActivityTypeBookingCreated,
		Description: "Free consultation booking created",
		Metadata:    "Booking ref: " + booking.BookingReference,
		CreatedAt:   time.Now(),
	}
	tx.Create(&activity)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("Failed to commit pre-talk booking transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process booking"})
		return
	}

	h.logger.Info("Free consultation booked successfully", 
		zap.String("booking_id", booking.ID.String()),
		zap.String("lead_id", lead.ID.String()),
		zap.String("email", req.Email),
		zap.String("timeslot_id", req.TimeslotID.String()))

	// TODO: Send confirmation email
	// TODO: Send notification to beraters

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Free consultation booked successfully",
		"booking_id":        booking.ID,
		"lead_id":           lead.ID,
		"booking_reference": booking.BookingReference,
		"timeslot":          timeslot,
	})
}

// GetContactForms handles listing contact forms (Admin/Berater only)
// @Summary List contact forms
// @Description Get list of contact form submissions
// @Tags contact
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /api/v1/contact/forms [get]
func (h *ContactHandler) GetContactForms(c *gin.Context) {
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Only beraters and admins can view contact forms
	if userRole != "berater" && userRole != "junior_berater" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// Parse pagination
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	offset := (page - 1) * limit

	status := c.Query("status")

	// Build query
	query := h.db.Model(&models.ContactForm{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get contact forms with preloaded relations
	var contactForms []models.ContactForm
	if err := query.Preload("User").Preload("Lead").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&contactForms).Error; err != nil {
		h.logger.Error("Failed to fetch contact forms", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contact forms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contact_forms": contactForms,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// UpdateContactFormStatus handles updating contact form status
// @Summary Update contact form status
// @Description Update the status of a contact form submission
// @Tags contact
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Contact Form ID"
// @Param request body map[string]interface{} true "Status update"
// @Success 200 {object} models.ContactForm
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/contact/forms/{id}/status [patch]
func (h *ContactHandler) UpdateContactFormStatus(c *gin.Context) {
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Only beraters and admins can update contact form status
	if userRole != "berater" && userRole != "junior_berater" && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	contactFormID := c.Param("id")

	var contactForm models.ContactForm
	if err := h.db.Where("id = ?", contactFormID).First(&contactForm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact form not found"})
		} else {
			h.logger.Error("Failed to fetch contact form", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contact form"})
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

	statusStr, ok := newStatus.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status format"})
		return
	}

	// Validate status
	validStatuses := []string{"new", "in_progress", "responded", "closed"}
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

	// Update status
	contactForm.Status = statusStr
	contactForm.UpdatedAt = time.Now()

	// Set response timestamp if marking as responded
	if statusStr == "responded" && contactForm.RespondedAt == nil {
		now := time.Now()
		contactForm.RespondedAt = &now
	}

	if err := h.db.Save(&contactForm).Error; err != nil {
		h.logger.Error("Failed to update contact form status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	h.logger.Info("Contact form status updated", 
		zap.String("contact_form_id", contactFormID),
		zap.String("new_status", statusStr))

	c.JSON(http.StatusOK, contactForm)
}