package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusCompleted BookingStatus = "completed"
	BookingStatusCancelled BookingStatus = "cancelled"
	BookingStatusNoShow    BookingStatus = "no_show"
)

type BookingType string

const (
	BookingTypeConsultation BookingType = "consultation"
	BookingTypePreTalk      BookingType = "pre_talk"
	BookingTypeFollowUp     BookingType = "follow_up"
)

// Booking represents a booked appointment
type Booking struct {
	ID        uuid.UUID     `json:"id" gorm:"type:char(36);primary_key"`
	UserID    uuid.UUID     `json:"user_id" gorm:"type:char(36);not null;index"`
	PackageID *uuid.UUID    `json:"package_id" gorm:"type:char(36);index"`
	BeraterID *uuid.UUID    `json:"berater_id" gorm:"type:char(36);index"`
	LeadID    *uuid.UUID    `json:"lead_id" gorm:"type:char(36);index"`
	PaymentID *uuid.UUID    `json:"payment_id" gorm:"type:char(36);index"`
	TimeslotID *uuid.UUID    `json:"timeslot_id" gorm:"type:char(36);index"`
	
	// Booking details
	Title       string        `json:"title" gorm:"not null" validate:"required"`
	Description string        `json:"description" gorm:"type:text"`
	Type        BookingType   `json:"type" gorm:"not null;default:'consultation'"`
	Status      BookingStatus `json:"status" gorm:"not null;default:'pending'"`
	
	// Timing
	ScheduledAt time.Time `json:"scheduled_at" gorm:"not null" validate:"required"`
	Duration    int       `json:"duration" gorm:"not null;default:60"` // in minutes
	StartTime   time.Time `json:"start_time" gorm:"not null"`
	EndTime     time.Time `json:"end_time" gorm:"not null"`
	
	// Contact information (filled after booking)
	CustomerName    string `json:"customer_name" gorm:""`
	CustomerEmail   string `json:"customer_email" gorm:""`
	CustomerPhone   string `json:"customer_phone" gorm:""`
	CustomerAddress string `json:"customer_address" gorm:"type:text"`
	CustomerNotes   string `json:"customer_notes" gorm:"type:text"`
	
	// Meeting details
	MeetingLink     string `json:"meeting_link" gorm:""`
	MeetingPassword string `json:"meeting_password" gorm:""`
	Location        string `json:"location" gorm:""`
	IsOnline        bool   `json:"is_online" gorm:"not null;default:true"`
	
	// Booking metadata
	BookingReference string `json:"booking_reference" gorm:"uniqueIndex"`
	InternalNotes    string `json:"internal_notes" gorm:"type:text"`
	CancellationNote string `json:"cancellation_note" gorm:"type:text"`
	
	// Pricing (for display purposes)
	TotalAmount float64 `json:"total_amount" gorm:"default:0"`
	Currency    string  `json:"currency" gorm:"default:'EUR'"`
	
	// Timestamps
	BookedAt     time.Time      `json:"booked_at" gorm:"not null"`
	ConfirmedAt  *time.Time     `json:"confirmed_at" gorm:""`
	CompletedAt  *time.Time     `json:"completed_at" gorm:""`
	CancelledAt  *time.Time     `json:"cancelled_at" gorm:""`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	User     User      `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Package  *Package  `json:"package,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Berater  *User     `json:"berater,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Lead     *Lead     `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Payment  *Payment  `json:"payment,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Timeslot *Timeslot `json:"timeslot,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Addons   []Addon   `json:"addons,omitempty" gorm:"many2many:booking_addons;"`
	Todos    []Todo    `json:"todos,omitempty" gorm:"foreignKey:BookingID"`
}

// BookingAddon represents the junction table for booking-addon relationships
type BookingAddon struct {
	BookingID uuid.UUID `json:"booking_id" gorm:"type:char(36);primary_key"`
	AddonID   uuid.UUID `json:"addon_id" gorm:"type:char(36);primary_key"`
	Price     float64   `json:"price" gorm:"not null"`
	
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	
	// Relationships
	Booking Booking `json:"booking,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Addon   Addon   `json:"addon,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Timeslot represents available time slots for beraters
type Timeslot struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	BeraterID uuid.UUID `json:"berater_id" gorm:"type:char(36);not null;index"`
	
	// Time details
	Date      time.Time `json:"date" gorm:"not null;index"`
	StartTime time.Time `json:"start_time" gorm:"not null"`
	EndTime   time.Time `json:"end_time" gorm:"not null"`
	Duration  int       `json:"duration" gorm:"not null"` // in minutes
	
	// Availability
	IsAvailable bool `json:"is_available" gorm:"not null;default:true"`
	IsRecurring bool `json:"is_recurring" gorm:"not null;default:false"`
	
	// Recurrence settings (if recurring)
	RecurrencePattern string    `json:"recurrence_pattern" gorm:""` // weekly, daily, etc.
	RecurrenceEnd     *time.Time `json:"recurrence_end" gorm:""`
	
	// Booking limits
	MaxBookings     int `json:"max_bookings" gorm:"not null;default:1"`
	CurrentBookings int `json:"current_bookings" gorm:"not null;default:0"`
	
	// Metadata
	Title       string `json:"title" gorm:""`
	Description string `json:"description" gorm:"type:text"`
	Location    string `json:"location" gorm:""`
	IsOnline    bool   `json:"is_online" gorm:"not null;default:true"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Berater  User      `json:"berater,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Bookings []Booking `json:"bookings,omitempty" gorm:"foreignKey:TimeslotID"`
}

// Todo represents tasks assigned to customers
type Todo struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	BookingID *uuid.UUID `json:"booking_id" gorm:"type:char(36);index"`
	LeadID    *uuid.UUID `json:"lead_id" gorm:"type:char(36);index"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"` // customer
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:char(36);not null;index"` // berater who created it
	
	// Todo details
	Title       string `json:"title" gorm:"not null" validate:"required"`
	Description string `json:"description" gorm:"type:text"`
	IsCompleted bool   `json:"is_completed" gorm:"not null;default:false"`
	
	// Timing
	DueDate     *time.Time `json:"due_date" gorm:""`
	CompletedAt *time.Time `json:"completed_at" gorm:""`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Booking   *Booking `json:"booking,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Lead      *Lead    `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	User      User     `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Creator   User     `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// BookingResponse represents the booking data returned in API responses
type BookingResponse struct {
	ID               uuid.UUID       `json:"id"`
	UserID           uuid.UUID       `json:"user_id"`
	PackageID        *uuid.UUID      `json:"package_id"`
	BeraterID        *uuid.UUID      `json:"berater_id"`
	LeadID           *uuid.UUID      `json:"lead_id"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Type             BookingType     `json:"type"`
	Status           BookingStatus   `json:"status"`
	ScheduledAt      time.Time       `json:"scheduled_at"`
	Duration         int             `json:"duration"`
	StartTime        time.Time       `json:"start_time"`
	EndTime          time.Time       `json:"end_time"`
	CustomerName     string          `json:"customer_name"`
	CustomerEmail    string          `json:"customer_email"`
	CustomerPhone    string          `json:"customer_phone"`
	MeetingLink      string          `json:"meeting_link"`
	Location         string          `json:"location"`
	IsOnline         bool            `json:"is_online"`
	BookingReference string          `json:"booking_reference"`
	TotalAmount      float64         `json:"total_amount"`
	FormattedAmount  string          `json:"formatted_amount"`
	Currency         string          `json:"currency"`
	BookedAt         time.Time       `json:"booked_at"`
	ConfirmedAt      *time.Time      `json:"confirmed_at"`
	CompletedAt      *time.Time      `json:"completed_at"`
	CancelledAt      *time.Time      `json:"cancelled_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	User             *UserResponse   `json:"user,omitempty"`
	Package          *PackageResponse `json:"package,omitempty"`
	Berater          *UserResponse   `json:"berater,omitempty"`
	SelectedAddons   []AddonResponse `json:"selected_addons,omitempty"`
	CanCancel        bool            `json:"can_cancel"`
	CanReschedule    bool            `json:"can_reschedule"`
}

// TimeslotResponse represents the timeslot data returned in API responses
type TimeslotResponse struct {
	ID              uuid.UUID    `json:"id"`
	BeraterID       uuid.UUID    `json:"berater_id"`
	Date            time.Time    `json:"date"`
	StartTime       time.Time    `json:"start_time"`
	EndTime         time.Time    `json:"end_time"`
	Duration        int          `json:"duration"`
	IsAvailable     bool         `json:"is_available"`
	MaxBookings     int          `json:"max_bookings"`
	CurrentBookings int          `json:"current_bookings"`
	AvailableSlots  int          `json:"available_slots"`
	Title           string       `json:"title"`
	Location        string       `json:"location"`
	IsOnline        bool         `json:"is_online"`
	Berater         *UserResponse `json:"berater,omitempty"`
}

// TodoResponse represents the todo data returned in API responses
type TodoResponse struct {
	ID          uuid.UUID    `json:"id"`
	BookingID   *uuid.UUID   `json:"booking_id"`
	LeadID      *uuid.UUID   `json:"lead_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	IsCompleted bool         `json:"is_completed"`
	DueDate     *time.Time   `json:"due_date"`
	CompletedAt *time.Time   `json:"completed_at"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Creator     *UserResponse `json:"creator,omitempty"`
}

// Request structs
type CreateBookingRequest struct {
	PackageID   uuid.UUID   `json:"package_id" validate:"required"`
	TimeslotID  *uuid.UUID  `json:"timeslot_id"`
	ScheduledAt time.Time   `json:"scheduled_at" validate:"required"`
	AddonIDs    []uuid.UUID `json:"addon_ids"`
	Notes       string      `json:"notes"`
}

type UpdateBookingContactRequest struct {
	CustomerName    string `json:"customer_name" validate:"required"`
	CustomerEmail   string `json:"customer_email" validate:"required,email"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerAddress string `json:"customer_address"`
	CustomerNotes   string `json:"customer_notes"`
}

type CreateTimeslotRequest struct {
	Date      time.Time `json:"date" validate:"required"`
	StartTime time.Time `json:"start_time" validate:"required"`
	EndTime   time.Time `json:"end_time" validate:"required"`
	Title     string    `json:"title"`
	Location  string    `json:"location"`
	IsOnline  bool      `json:"is_online"`
}

type CreateTodoRequest struct {
	BookingID   *uuid.UUID `json:"booking_id"`
	LeadID      *uuid.UUID `json:"lead_id"`
	UserID      uuid.UUID  `json:"user_id" validate:"required"`
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date"`
}

type UpdateTodoRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	IsCompleted *bool      `json:"is_completed"`
	DueDate     *time.Time `json:"due_date"`
}

// BeforeCreate hooks
func (b *Booking) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.BookingReference == "" {
		b.BookingReference = b.generateBookingReference()
	}
	if b.Currency == "" {
		b.Currency = "EUR"
	}
	
	// Set booking time as start time if not provided
	if b.StartTime.IsZero() {
		b.StartTime = b.ScheduledAt
	}
	if b.EndTime.IsZero() {
		b.EndTime = b.StartTime.Add(time.Duration(b.Duration) * time.Minute)
	}
	if b.BookedAt.IsZero() {
		b.BookedAt = time.Now()
	}
	
	return nil
}

func (t *Timeslot) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (td *Todo) BeforeCreate(tx *gorm.DB) error {
	if td.ID == uuid.Nil {
		td.ID = uuid.New()
	}
	return nil
}

// Helper methods
func (b *Booking) generateBookingReference() string {
	shortID := b.ID.String()[:8]
	return fmt.Sprintf("BK-%d-%s", time.Now().Year(), shortID)
}

func (b *Booking) ToResponse() BookingResponse {
	response := BookingResponse{
		ID:               b.ID,
		UserID:           b.UserID,
		PackageID:        b.PackageID,
		BeraterID:        b.BeraterID,
		LeadID:           b.LeadID,
		Title:            b.Title,
		Description:      b.Description,
		Type:             b.Type,
		Status:           b.Status,
		ScheduledAt:      b.ScheduledAt,
		Duration:         b.Duration,
		StartTime:        b.StartTime,
		EndTime:          b.EndTime,
		CustomerName:     b.CustomerName,
		CustomerEmail:    b.CustomerEmail,
		CustomerPhone:    b.CustomerPhone,
		MeetingLink:      b.MeetingLink,
		Location:         b.Location,
		IsOnline:         b.IsOnline,
		BookingReference: b.BookingReference,
		TotalAmount:      b.TotalAmount,
		FormattedAmount:  b.FormatAmount(),
		Currency:         b.Currency,
		BookedAt:         b.BookedAt,
		ConfirmedAt:      b.ConfirmedAt,
		CompletedAt:      b.CompletedAt,
		CancelledAt:      b.CancelledAt,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		CanCancel:        b.CanCancel(),
		CanReschedule:    b.CanReschedule(),
	}
	
	// Add relationships
	if b.User.ID != uuid.Nil {
		userResponse := b.User.ToResponse()
		response.User = &userResponse
	}
	
	if b.Package != nil {
		packageResponse := b.Package.ToResponse()
		response.Package = &packageResponse
	}
	
	if b.Berater != nil && b.Berater.ID != uuid.Nil {
		beraterResponse := b.Berater.ToResponse()
		response.Berater = &beraterResponse
	}
	
	// Add selected addons
	for _, addon := range b.Addons {
		response.SelectedAddons = append(response.SelectedAddons, addon.ToResponse())
	}
	
	return response
}

func (t *Timeslot) ToResponse() TimeslotResponse {
	response := TimeslotResponse{
		ID:              t.ID,
		BeraterID:       t.BeraterID,
		Date:            t.Date,
		StartTime:       t.StartTime,
		EndTime:         t.EndTime,
		Duration:        t.Duration,
		IsAvailable:     t.IsAvailable,
		MaxBookings:     t.MaxBookings,
		CurrentBookings: t.CurrentBookings,
		AvailableSlots:  t.MaxBookings - t.CurrentBookings,
		Title:           t.Title,
		Location:        t.Location,
		IsOnline:        t.IsOnline,
	}
	
	if t.Berater.ID != uuid.Nil {
		beraterResponse := t.Berater.ToResponse()
		response.Berater = &beraterResponse
	}
	
	return response
}

func (td *Todo) ToResponse() TodoResponse {
	response := TodoResponse{
		ID:          td.ID,
		BookingID:   td.BookingID,
		LeadID:      td.LeadID,
		Title:       td.Title,
		Description: td.Description,
		IsCompleted: td.IsCompleted,
		DueDate:     td.DueDate,
		CompletedAt: td.CompletedAt,
		CreatedAt:   td.CreatedAt,
		UpdatedAt:   td.UpdatedAt,
	}
	
	if td.Creator.ID != uuid.Nil {
		creatorResponse := td.Creator.ToResponse()
		response.Creator = &creatorResponse
	}
	
	return response
}

// Utility methods
func (b *Booking) FormatAmount() string {
	return formatCurrency(b.TotalAmount, b.Currency)
}

func (b *Booking) CanCancel() bool {
	// Can cancel if booking is pending or confirmed and not in the past
	if b.Status != BookingStatusPending && b.Status != BookingStatusConfirmed {
		return false
	}
	return time.Now().Before(b.StartTime.Add(-24 * time.Hour)) // 24h before appointment
}

func (b *Booking) CanReschedule() bool {
	// Can reschedule if booking is pending or confirmed and not in the past
	if b.Status != BookingStatusPending && b.Status != BookingStatusConfirmed {
		return false
	}
	return time.Now().Before(b.StartTime.Add(-24 * time.Hour)) // 24h before appointment
}

func (b *Booking) IsUpcoming() bool {
	return time.Now().Before(b.StartTime)
}

func (b *Booking) IsOverdue() bool {
	return time.Now().After(b.EndTime) && b.Status == BookingStatusConfirmed
}

func (t *Timeslot) HasAvailableSlots() bool {
	return t.IsAvailable && t.CurrentBookings < t.MaxBookings
}

func (t *Timeslot) IsInPast() bool {
	return time.Now().After(t.EndTime)
}

func (td *Todo) MarkCompleted() {
	td.IsCompleted = true
	now := time.Now()
	td.CompletedAt = &now
}

func (td *Todo) IsOverdue() bool {
	if td.DueDate == nil || td.IsCompleted {
		return false
	}
	return time.Now().After(*td.DueDate)
}

// Status helper methods
func (bs BookingStatus) GetDisplayName() string {
	switch bs {
	case BookingStatusPending:
		return "Ausstehend"
	case BookingStatusConfirmed:
		return "Bestätigt"
	case BookingStatusCompleted:
		return "Abgeschlossen"
	case BookingStatusCancelled:
		return "Storniert"
	case BookingStatusNoShow:
		return "Nicht erschienen"
	default:
		return "Unbekannt"
	}
}

func (bt BookingType) GetDisplayName() string {
	switch bt {
	case BookingTypeConsultation:
		return "Beratung"
	case BookingTypePreTalk:
		return "Vorgespräch"
	case BookingTypeFollowUp:
		return "Nachtermin"
	default:
		return "Unbekannt"
	}
}