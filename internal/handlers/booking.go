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

type BookingHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBookingHandler(db *gorm.DB, logger *zap.Logger) *BookingHandler {
	return &BookingHandler{
		db:     db,
		logger: logger,
	}
}

// CreateBookingRequest represents the booking creation request
type CreateBookingRequest struct {
	PackageID     uuid.UUID   `json:"package_id" binding:"required"`
	AddOnIDs      []uuid.UUID `json:"addon_ids,omitempty"`
	TimeslotID    *uuid.UUID  `json:"timeslot_id,omitempty"`
	PreferredDate *time.Time  `json:"preferred_date,omitempty"`
	Notes         string      `json:"notes,omitempty"`
}

// UpdateContactInfoRequest represents the contact info update after booking
type UpdateContactInfoRequest struct {
	FirstName     string `json:"first_name" binding:"required"`
	LastName      string `json:"last_name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	Street        string `json:"street" binding:"required"`
	HouseNumber   string `json:"house_number" binding:"required"`
	PostalCode    string `json:"postal_code" binding:"required"`
	City          string `json:"city" binding:"required"`
	Country       string `json:"country" binding:"required"`
	DateOfBirth   string `json:"date_of_birth,omitempty"`
	PartnerName   string `json:"partner_name,omitempty"`
	ChildrenCount int    `json:"children_count,omitempty"`
}

// BookingResponse represents a booking with related data
type BookingResponse struct {
	*models.Booking
	Package   *models.Package    `json:"package,omitempty"`
	AddOns    []models.Package   `json:"addons,omitempty"`
	Timeslot  *models.Timeslot   `json:"timeslot,omitempty"`
	Lead      *models.Lead       `json:"lead,omitempty"`
	Payments  []models.Payment   `json:"payments,omitempty"`
	Documents []models.Document  `json:"documents,omitempty"`
}

// ListPackages handles listing available packages for pricing page
// @Summary List packages
// @Description Get list of available service packages for pricing page
// @Tags packages
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/packages [get]
func (h *BookingHandler) ListPackages(c *gin.Context) {
	var packages []models.Package
	if err := h.db.Where("category = ? AND is_active = ?", "service", true).
		Order("sort_order ASC, price ASC").Find(&packages).Error; err != nil {
		h.logger.Error("Failed to fetch packages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch packages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"packages": packages,
	})
}

// GetPackageAddOns handles getting add-ons for a specific package
// @Summary Get package add-ons
// @Description Get available add-ons for a specific package
// @Tags packages
// @Produce json
// @Param id path string true "Package ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/packages/{id}/addons [get]
func (h *BookingHandler) GetPackageAddOns(c *gin.Context) {
	packageID := c.Param("id")

	// Verify package exists
	var servicePackage models.Package
	if err := h.db.Where("id = ? AND type = ?", packageID, "service").First(&servicePackage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		} else {
			h.logger.Error("Failed to fetch package", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch package"})
		}
		return
	}

	// Get add-ons
	var addOns []models.Package
	if err := h.db.Where("type = ? AND is_active = ?", "addon", true).
		Order("sort_order ASC, price ASC").Find(&addOns).Error; err != nil {
		h.logger.Error("Failed to fetch add-ons", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch add-ons"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"package": servicePackage,
		"addons":  addOns,
	})
}

// GetAvailableTimeslots handles getting available timeslots for booking
// @Summary Get available timeslots
// @Description Get available timeslots for a package (if required)
// @Tags timeslots
// @Produce json
// @Param package_id query string true "Package ID"
// @Param date query string false "Date (YYYY-MM-DD)"
// @Param days query int false "Number of days to look ahead (default: 30)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/timeslots/available [get]
func (h *BookingHandler) GetAvailableTimeslots(c *gin.Context) {
	packageID := c.Query("package_id")
	if packageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Package ID is required"})
		return
	}

	// Verify package exists and get booking details
	var servicePackage models.Package
	if err := h.db.Where("id = ?", packageID).First(&servicePackage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		} else {
			h.logger.Error("Failed to fetch package", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch package"})
		}
		return
	}

	// If package doesn't require timeslot booking, return empty
	if !servicePackage.RequiresTimeslot {
		c.JSON(http.StatusOK, gin.H{
			"package":   servicePackage,
			"timeslots": []models.Timeslot{},
			"message":   "This package does not require timeslot selection",
		})
		return
	}

	// Parse date and days parameters
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	startDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	endDate := startDate.AddDate(0, 0, days)

	// Get available timeslots
	var timeslots []models.Timeslot
	query := h.db.Where("date_time >= ? AND date_time <= ? AND is_available = ?", 
		startDate, endDate, true)

	// If package has duration, filter by compatible timeslots
	if servicePackage.DurationMinutes > 0 {
		query = query.Where("duration_minutes >= ?", servicePackage.DurationMinutes)
	}

	if err := query.Order("date_time ASC").Find(&timeslots).Error; err != nil {
		h.logger.Error("Failed to fetch timeslots", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch timeslots"})
		return
	}

	// Check current bookings to filter out unavailable slots
	availableTimeslots := []models.Timeslot{}
	for _, slot := range timeslots {
		var bookingCount int64
		h.db.Model(&models.Booking{}).Where("timeslot_id = ? AND status NOT IN (?)", 
			slot.ID, []string{"cancelled", "completed"}).Count(&bookingCount)
		
		if bookingCount < int64(slot.MaxBookings) {
			availableTimeslots = append(availableTimeslots, slot)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"package":   servicePackage,
		"timeslots": availableTimeslots,
		"period": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
	})
}

// CreateBooking handles creating a new booking
// @Summary Create booking
// @Description Create a new booking with package, add-ons and optional timeslot
// @Tags bookings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateBookingRequest true "Booking data"
// @Success 201 {object} BookingResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/bookings [post]
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Verify package exists
	var servicePackage models.Package
	if err := tx.Where("id = ? AND type = ?", req.PackageID, "service").First(&servicePackage).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		} else {
			h.logger.Error("Failed to fetch package", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch package"})
		}
		return
	}

	// Verify add-ons if provided
	var addOns []models.Package
	totalPrice := servicePackage.Price
	if len(req.AddOnIDs) > 0 {
		if err := tx.Where("id IN ? AND type = ?", req.AddOnIDs, "addon").Find(&addOns).Error; err != nil {
			tx.Rollback()
			h.logger.Error("Failed to fetch add-ons", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch add-ons"})
			return
		}

		if len(addOns) != len(req.AddOnIDs) {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "One or more add-ons not found"})
			return
		}

		// Calculate total price
		for _, addOn := range addOns {
			totalPrice += addOn.Price
		}
	}

	// Verify timeslot if provided
	var timeslot *models.Timeslot
	if req.TimeslotID != nil {
		timeslot = &models.Timeslot{}
		if err := tx.Where("id = ? AND is_available = ?", *req.TimeslotID, true).First(timeslot).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Timeslot not found or not available"})
			} else {
				h.logger.Error("Failed to fetch timeslot", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch timeslot"})
			}
			return
		}

		// Check if timeslot is still available
		var bookingCount int64
		tx.Model(&models.Booking{}).Where("timeslot_id = ? AND status NOT IN (?)", 
			timeslot.ID, []string{"cancelled", "completed"}).Count(&bookingCount)
		
		if bookingCount >= int64(timeslot.MaxBookings) {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "Timeslot is no longer available"})
			return
		}
	} else if servicePackage.RequiresTimeslot {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "This package requires timeslot selection"})
		return
	}

	// Generate booking reference
	bookingRef := "BK" + time.Now().Format("20060102") + "-" + uuid.New().String()[:8]

	// Create booking
	booking := models.Booking{
		ID:               uuid.New(),
		UserID:           userID.(uuid.UUID),
		PackageID:        &req.PackageID,
		TimeslotID:       req.TimeslotID,
		BookingReference: bookingRef,
		Status:           models.BookingStatusPending,
		TotalAmount:       totalPrice,
		Currency:         "EUR",
		BookedAt:      time.Now(),
		CustomerNotes:            req.Notes,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := tx.Create(&booking).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to create booking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Create booking add-ons
	for _, addOn := range addOns {
		bookingAddOn := models.BookingAddon{
			BookingID:        booking.ID,
			AddonID: addOn.ID,
			Price:     addOn.Price,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&bookingAddOn).Error; err != nil {
			tx.Rollback()
			h.logger.Error("Failed to create booking add-on", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
			return
		}
	}

	// Create associated lead
	lead := models.Lead{
		ID:           uuid.New(),
		UserID:       userID.(uuid.UUID),
		Source:       models.LeadSourceBooking,
		Status:       models.LeadStatusNew,
		Priority:     models.PriorityMedium,
		EstimatedValue: totalPrice,
		Title:        "Booking: " + servicePackage.Name,
		Description:  "New booking created for " + servicePackage.Name,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := tx.Create(&lead).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to create lead", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Update booking with lead ID
	booking.LeadID = &lead.ID
	if err := tx.Save(&booking).Error; err != nil {
		tx.Rollback()
		h.logger.Error("Failed to update booking with lead ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("Failed to commit booking transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	h.logger.Info("Booking created successfully", 
		zap.String("booking_id", booking.ID.String()),
		zap.String("user_id", userID.(uuid.UUID).String()),
		zap.String("package_id", req.PackageID.String()))

	// Prepare response
	response := &BookingResponse{
		Booking:  &booking,
		Package:  &servicePackage,
		AddOns:   addOns,
		Timeslot: timeslot,
		Lead:     &lead,
	}

	c.JSON(http.StatusCreated, response)
}

// GetUserBookings handles listing user's bookings
// @Summary Get user bookings
// @Description Get list of current user's bookings
// @Tags bookings
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/bookings [get]
func (h *BookingHandler) GetUserBookings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	status := c.Query("status")

	// Build query
	query := h.db.Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var total int64
	query.Model(&models.Booking{}).Count(&total)

	// Get bookings with preloaded relations
	var bookings []models.Booking
	if err := query.Preload("Package").Preload("Timeslot").Preload("Lead").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&bookings).Error; err != nil {
		h.logger.Error("Failed to fetch user bookings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bookings": bookings,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetBooking handles getting a specific booking
// @Summary Get booking by ID
// @Description Get booking details with all related data
// @Tags bookings
// @Security BearerAuth
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} BookingResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/bookings/{id} [get]
func (h *BookingHandler) GetBooking(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	bookingID := c.Param("id")

	var booking models.Booking
	query := h.db.Where("id = ?", bookingID)

	// Non-admin users can only see their own bookings
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && userRole != "berater" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Preload("Package").Preload("Timeslot").Preload("Lead").
		Preload("Payments").Preload("Documents").First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		} else {
			h.logger.Error("Failed to fetch booking", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booking"})
		}
		return
	}

	// Get add-ons
	var addOns []models.Package
	h.db.Table("booking_add_ons").
		Select("packages.*").
		Joins("JOIN packages ON packages.id = booking_add_ons.package_id").
		Where("booking_add_ons.booking_id = ?", booking.ID).
		Find(&addOns)

	response := &BookingResponse{
		Booking:   &booking,
		Package:   booking.Package,
		AddOns:    addOns,
		Timeslot:  booking.Timeslot,
		Lead:      booking.Lead,
		Payments:  nil,
		Documents: nil,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateBookingContactInfo handles updating contact information after booking
// @Summary Update booking contact info
// @Description Update contact information for a booking (must be done after booking)
// @Tags bookings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Param request body UpdateContactInfoRequest true "Contact information"
// @Success 200 {object} models.Booking
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/bookings/{id}/contact-info [put]
func (h *BookingHandler) UpdateBookingContactInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	bookingID := c.Param("id")

	var booking models.Booking
	if err := h.db.Where("id = ? AND user_id = ?", bookingID, userID).First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		} else {
			h.logger.Error("Failed to fetch booking", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booking"})
		}
		return
	}

	var req UpdateContactInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Update booking contact information
	updates := map[string]interface{}{
		"contact_first_name":  req.FirstName,
		"contact_last_name":   req.LastName,
		"contact_phone":       req.Phone,
		"contact_street":      req.Street,
		"contact_house_number": req.HouseNumber,
		"contact_postal_code": req.PostalCode,
		"contact_city":        req.City,
		"contact_country":     req.Country,
		"contact_completed":   true,
		"updated_at":          time.Now(),
	}

	if req.DateOfBirth != "" {
		updates["contact_date_of_birth"] = req.DateOfBirth
	}
	if req.PartnerName != "" {
		updates["contact_partner_name"] = req.PartnerName
	}
	if req.ChildrenCount > 0 {
		updates["contact_children_count"] = req.ChildrenCount
	}

	if err := h.db.Model(&booking).Updates(updates).Error; err != nil {
		h.logger.Error("Failed to update contact info", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contact information"})
		return
	}

	// Fetch updated booking
	if err := h.db.First(&booking, "id = ?", bookingID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated booking"})
		return
	}

	h.logger.Info("Contact info updated", zap.String("booking_id", bookingID))

	c.JSON(http.StatusOK, booking)
}