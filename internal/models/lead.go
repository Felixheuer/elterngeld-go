package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeadPriority represents the priority level of a lead
type LeadPriority string

const (
	LeadPriorityLow    LeadPriority = "low"
	LeadPriorityMedium LeadPriority = "medium"
	LeadPriorityHigh   LeadPriority = "high"
	LeadPriorityCritical LeadPriority = "critical"
)

type LeadStatus string

const (
	LeadStatusNew            LeadStatus = "neu"
	LeadStatusInProgress     LeadStatus = "in_bearbeitung"
	LeadStatusQuestion       LeadStatus = "rückfrage"
	LeadStatusCompleted      LeadStatus = "abgeschlossen"
	LeadStatusCancelled      LeadStatus = "storniert"
	LeadStatusPaymentPending LeadStatus = "zahlung_ausstehend"
	LeadStatusQualified      LeadStatus = "qualifiziert"
	LeadStatusUnqualified    LeadStatus = "nicht_qualifiziert"
)

type Priority string

const (
	PriorityLow    Priority = "niedrig"
	PriorityMedium Priority = "mittel"
	PriorityHigh   Priority = "hoch"
	PriorityUrgent Priority = "dringend"
)

type LeadSource string

const (
	LeadSourceWebsite     LeadSource = "website"
	LeadSourceBooking     LeadSource = "booking"
	LeadSourceContact     LeadSource = "contact_form"
	LeadSourceReferral    LeadSource = "referral"
	LeadSourcePhone       LeadSource = "phone"
	LeadSourceEmail       LeadSource = "email"
	LeadSourceSocial      LeadSource = "social_media"
	LeadSourceManual      LeadSource = "manual"
)

// Lead represents an Elterngeld application/case
type Lead struct {
	ID        uuid.UUID  `json:"id" gorm:"type:char(36);primary_key"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:char(36);not null;index"`
	BeraterID *uuid.UUID `json:"berater_id" gorm:"type:char(36);index"`

	// Lead information
	Title       string     `json:"title" gorm:"not null" validate:"required"`
	Description string     `json:"description" gorm:"type:text"`
	Status      LeadStatus `json:"status" gorm:"not null;default:'neu'" validate:"required"`
	Priority    Priority   `json:"priority" gorm:"not null;default:'mittel'" validate:"required"`

	// Lead source and tracking
	Source          LeadSource `json:"source" gorm:"not null;default:'manual'" validate:"required"`
	SourceDetails   string     `json:"source_details" gorm:"type:text"`
	ReferralSource  string     `json:"referral_source" gorm:""`
	UtmSource       string     `json:"utm_source" gorm:""`
	UtmMedium       string     `json:"utm_medium" gorm:""`
	UtmCampaign     string     `json:"utm_campaign" gorm:""`
	
	// Contact attempt tracking
	ContactAttempts     int        `json:"contact_attempts" gorm:"default:0"`
	LastContactAt       *time.Time `json:"last_contact_at" gorm:""`
	NextFollowUpAt      *time.Time `json:"next_follow_up_at" gorm:""`
	NextFollowUpNote    string     `json:"next_follow_up_note" gorm:"type:text"`
	
	// Qualification
	IsQualified         bool   `json:"is_qualified" gorm:"not null;default:false"`
	QualificationNotes  string `json:"qualification_notes" gorm:"type:text"`
	QualifiedAt         *time.Time `json:"qualified_at" gorm:""`
	
	// Value estimation
	EstimatedValue      float64 `json:"estimated_value" gorm:"default:0"`
	EstimatedCloseDate  *time.Time `json:"estimated_close_date" gorm:""`
	
	// Communication preferences
	PreferredContactMethod string `json:"preferred_contact_method" gorm:"default:'email'"` // email, phone, both
	PreferredContactTime   string `json:"preferred_contact_time" gorm:""` // morning, afternoon, evening
	Timezone              string `json:"timezone" gorm:"default:'Europe/Berlin'"`
	
	// Lead scoring
	LeadScore          int    `json:"lead_score" gorm:"default:0"`
	LeadScoreReason    string `json:"lead_score_reason" gorm:"type:text"`
	
	// Conversion tracking
	ConvertedAt        *time.Time `json:"converted_at" gorm:""`
	ConversionValue    float64    `json:"conversion_value" gorm:"default:0"`
	
	// Elterngeld specific fields
	ChildName         string     `json:"child_name" gorm:""`
	ChildBirthDate    *time.Time `json:"child_birth_date" gorm:""`
	ExpectedAmount    float64    `json:"expected_amount" gorm:""`
	ApplicationNumber string     `json:"application_number" gorm:"uniqueIndex"`

	// Contact preferences
	PreferredContact string `json:"preferred_contact" gorm:"default:'email'"` // email, phone, both

	// Timeline
	DueDate     *time.Time `json:"due_date" gorm:""`
	CompletedAt *time.Time `json:"completed_at" gorm:""`

	// Internal notes
	InternalNotes string `json:"internal_notes" gorm:"type:text"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User         User            `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Berater      *User           `json:"berater,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Documents    []Document      `json:"documents,omitempty" gorm:"foreignKey:LeadID"`
	Activities   []Activity      `json:"activities,omitempty" gorm:"foreignKey:LeadID"`
	Payments     []Payment       `json:"payments,omitempty" gorm:"foreignKey:LeadID"`
	Comments     []Comment       `json:"comments,omitempty" gorm:"foreignKey:LeadID"`
	Bookings     []Booking       `json:"bookings,omitempty" gorm:"foreignKey:LeadID"`
	Todos        []Todo          `json:"todos,omitempty" gorm:"foreignKey:LeadID"`
	Reminders    []Reminder      `json:"reminders,omitempty" gorm:"foreignKey:LeadID"`
	EmailThreads []EmailThread   `json:"email_threads,omitempty" gorm:"foreignKey:LeadID"`
}

// Reminder represents follow-up reminders for leads
type Reminder struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	LeadID    uuid.UUID `json:"lead_id" gorm:"type:char(36);not null;index"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"` // who should be reminded
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:char(36);not null;index"` // who created the reminder
	
	Title       string    `json:"title" gorm:"not null" validate:"required"`
	Description string    `json:"description" gorm:"type:text"`
	RemindAt    time.Time `json:"remind_at" gorm:"not null" validate:"required"`
	IsCompleted bool      `json:"is_completed" gorm:"not null;default:false"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Lead    Lead `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User    User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Creator User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// EmailThread represents email communication threads for leads
type EmailThread struct {
	ID           uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	LeadID       uuid.UUID `json:"lead_id" gorm:"type:char(36);not null;index"`
	Subject      string    `json:"subject" gorm:"not null"`
	ThreadID     string    `json:"thread_id" gorm:"uniqueIndex"` // External email thread ID
	LastMessageAt time.Time `json:"last_message_at" gorm:"not null"`
	MessageCount  int       `json:"message_count" gorm:"default:0"`
	IsActive      bool      `json:"is_active" gorm:"not null;default:true"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Lead     Lead          `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Messages []EmailMessage `json:"messages,omitempty" gorm:"foreignKey:ThreadID"`
}

// EmailMessage represents individual email messages in a thread
type EmailMessage struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	ThreadID  uuid.UUID `json:"thread_id" gorm:"type:char(36);not null;index"`
	MessageID string    `json:"message_id" gorm:"uniqueIndex"` // External message ID
	
	FromEmail string `json:"from_email" gorm:"not null"`
	ToEmail   string `json:"to_email" gorm:"not null"`
	Subject   string `json:"subject" gorm:"not null"`
	Body      string `json:"body" gorm:"type:text"`
	IsHTML    bool   `json:"is_html" gorm:"not null;default:false"`
	
	IsInbound bool `json:"is_inbound" gorm:"not null;default:true"` // true if from customer
	IsRead    bool `json:"is_read" gorm:"not null;default:false"`
	
	SentAt    time.Time `json:"sent_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null"`
	
	// Relationships
	Thread EmailThread `json:"thread,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Comment represents comments on a lead
type Comment struct {
	ID         uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	LeadID     uuid.UUID `json:"lead_id" gorm:"type:char(36);not null;index"`
	UserID     uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	Content    string    `json:"content" gorm:"type:text;not null" validate:"required"`
	IsInternal bool      `json:"is_internal" gorm:"not null;default:false"`

	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Lead Lead `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// CreateLeadRequest represents the request body for creating a lead
type CreateLeadRequest struct {
	Title            string     `json:"title" validate:"required"`
	Description      string     `json:"description"`
	ChildName        string     `json:"child_name"`
	ChildBirthDate   *time.Time `json:"child_birth_date"`
	ExpectedAmount   float64    `json:"expected_amount"`
	PreferredContact string     `json:"preferred_contact" validate:"omitempty,oneof=email phone both"`
	DueDate          *time.Time `json:"due_date"`
}

// UpdateLeadRequest represents the request body for updating a lead
type UpdateLeadRequest struct {
	Title            *string    `json:"title"`
	Description      *string    `json:"description"`
	Priority         *Priority  `json:"priority" validate:"omitempty,oneof=niedrig mittel hoch dringend"`
	ChildName        *string    `json:"child_name"`
	ChildBirthDate   *time.Time `json:"child_birth_date"`
	ExpectedAmount   *float64   `json:"expected_amount"`
	PreferredContact *string    `json:"preferred_contact" validate:"omitempty,oneof=email phone both"`
	DueDate          *time.Time `json:"due_date"`
	InternalNotes    *string    `json:"internal_notes"`
}

// UpdateLeadStatusRequest represents the request body for updating lead status
type UpdateLeadStatusRequest struct {
	Status  LeadStatus `json:"status" validate:"required,oneof=neu in_bearbeitung rückfrage abgeschlossen storniert zahlung_ausstehend"`
	Comment string     `json:"comment"`
}

// AssignLeadRequest represents the request body for assigning a lead to a berater
type AssignLeadRequest struct {
	BeraterID uuid.UUID `json:"berater_id" validate:"required"`
}

// CreateCommentRequest represents the request body for creating a comment
type CreateCommentRequest struct {
	Content    string `json:"content" validate:"required"`
	IsInternal bool   `json:"is_internal"`
}

// LeadResponse represents the lead data returned in API responses
type LeadResponse struct {
	ID                uuid.UUID     `json:"id"`
	UserID            uuid.UUID     `json:"user_id"`
	BeraterID         *uuid.UUID    `json:"berater_id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Status            LeadStatus    `json:"status"`
	Priority          Priority      `json:"priority"`
	ChildName         string        `json:"child_name"`
	ChildBirthDate    *time.Time    `json:"child_birth_date"`
	ExpectedAmount    float64       `json:"expected_amount"`
	ApplicationNumber string        `json:"application_number"`
	PreferredContact  string        `json:"preferred_contact"`
	DueDate           *time.Time    `json:"due_date"`
	CompletedAt       *time.Time    `json:"completed_at"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	User              *UserResponse `json:"user,omitempty"`
	Berater           *UserResponse `json:"berater,omitempty"`
	DocumentCount     int           `json:"document_count"`
	CommentCount      int           `json:"comment_count"`
}

// BeforeCreate is a GORM hook that runs before creating a lead
func (l *Lead) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}

	// Generate application number if not provided
	if l.ApplicationNumber == "" {
		l.ApplicationNumber = l.generateApplicationNumber()
	}

	return nil
}

// generateApplicationNumber generates a unique application number
func (l *Lead) generateApplicationNumber() string {
	// Format: EG-YYYY-XXXXXX (EG = Elterngeld, YYYY = Year, XXXXXX = Random)
	year := time.Now().Year()
	shortID := l.ID.String()[:6]
	return fmt.Sprintf("EG-%d-%s", year, strings.ToUpper(shortID))
}

// ToResponse converts a Lead to LeadResponse
func (l *Lead) ToResponse() LeadResponse {
	response := LeadResponse{
		ID:                l.ID,
		UserID:            l.UserID,
		BeraterID:         l.BeraterID,
		Title:             l.Title,
		Description:       l.Description,
		Status:            l.Status,
		Priority:          l.Priority,
		ChildName:         l.ChildName,
		ChildBirthDate:    l.ChildBirthDate,
		ExpectedAmount:    l.ExpectedAmount,
		ApplicationNumber: l.ApplicationNumber,
		PreferredContact:  l.PreferredContact,
		DueDate:           l.DueDate,
		CompletedAt:       l.CompletedAt,
		CreatedAt:         l.CreatedAt,
		UpdatedAt:         l.UpdatedAt,
		DocumentCount:     len(l.Documents),
		CommentCount:      len(l.Comments),
	}

	if l.User.ID != uuid.Nil {
		userResponse := l.User.ToResponse()
		response.User = &userResponse
	}

	if l.Berater != nil && l.Berater.ID != uuid.Nil {
		beraterResponse := l.Berater.ToResponse()
		response.Berater = &beraterResponse
	}

	return response
}

// CanTransitionTo checks if the lead can transition to the specified status
func (l *Lead) CanTransitionTo(newStatus LeadStatus) bool {
	allowedTransitions := map[LeadStatus][]LeadStatus{
		LeadStatusNew: {
			LeadStatusInProgress,
			LeadStatusPaymentPending,
			LeadStatusCancelled,
		},
		LeadStatusPaymentPending: {
			LeadStatusInProgress,
			LeadStatusCancelled,
		},
		LeadStatusInProgress: {
			LeadStatusQuestion,
			LeadStatusCompleted,
			LeadStatusCancelled,
		},
		LeadStatusQuestion: {
			LeadStatusInProgress,
			LeadStatusCompleted,
			LeadStatusCancelled,
		},
		LeadStatusCompleted: {
			// No transitions allowed from completed
		},
		LeadStatusCancelled: {
			LeadStatusNew,
			LeadStatusInProgress,
		},
	}

	allowed, exists := allowedTransitions[l.Status]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}

	return false
}

// IsCompleted checks if the lead is completed
func (l *Lead) IsCompleted() bool {
	return l.Status == LeadStatusCompleted
}

// IsCancelled checks if the lead is cancelled
func (l *Lead) IsCancelled() bool {
	return l.Status == LeadStatusCancelled
}

// IsActive checks if the lead is active (not completed or cancelled)
func (l *Lead) IsActive() bool {
	return !l.IsCompleted() && !l.IsCancelled()
}

// NeedsPayment checks if the lead needs payment
func (l *Lead) NeedsPayment() bool {
	return l.Status == LeadStatusPaymentPending
}

// IsOverdue checks if the lead is overdue
func (l *Lead) IsOverdue() bool {
	if l.DueDate == nil {
		return false
	}
	return time.Now().After(*l.DueDate) && l.IsActive()
}
