package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationTypeEmail     NotificationType = "email"
	NotificationTypeSMS       NotificationType = "sms"
	NotificationTypeInApp     NotificationType = "in_app"
	NotificationTypePush      NotificationType = "push"
)

type NotificationStatus string

const (
	NotificationStatusPending    NotificationStatus = "pending"
	NotificationStatusSent       NotificationStatus = "sent"
	NotificationStatusDelivered  NotificationStatus = "delivered"
	NotificationStatusFailed     NotificationStatus = "failed"
	NotificationStatusRetrying   NotificationStatus = "retrying"
)

type EmailTemplate string

const (
	EmailTemplateWelcome              EmailTemplate = "welcome"
	EmailTemplateEmailVerification    EmailTemplate = "email_verification"
	EmailTemplatePasswordReset        EmailTemplate = "password_reset"
	EmailTemplateBookingConfirmation  EmailTemplate = "booking_confirmation"
	EmailTemplateBookingReminder      EmailTemplate = "booking_reminder"
	EmailTemplateBookingCancellation  EmailTemplate = "booking_cancellation"
	EmailTemplateOrderConfirmation    EmailTemplate = "order_confirmation"
	EmailTemplatePaymentReceived      EmailTemplate = "payment_received"
	EmailTemplatePaymentFailed        EmailTemplate = "payment_failed"
	EmailTemplateTodoAssigned         EmailTemplate = "todo_assigned"
	EmailTemplateLeadAssigned         EmailTemplate = "lead_assigned"
	EmailTemplateReminderDue          EmailTemplate = "reminder_due"
	EmailTemplateContactForm          EmailTemplate = "contact_form"
)

// Notification represents a notification to be sent to a user
type Notification struct {
	ID       uuid.UUID        `json:"id" gorm:"type:char(36);primary_key"`
	UserID   uuid.UUID        `json:"user_id" gorm:"type:char(36);not null;index"`
	Type     NotificationType `json:"type" gorm:"not null"`
	Status   NotificationStatus `json:"status" gorm:"not null;default:'pending'"`
	
	// Content
	Title    string `json:"title" gorm:"not null"`
	Message  string `json:"message" gorm:"type:text;not null"`
	Data     string `json:"data" gorm:"type:text"` // JSON data for additional context
	
	// Template information
	Template     string `json:"template" gorm:""`
	TemplateData string `json:"template_data" gorm:"type:text"` // JSON data for template variables
	
	// Recipients
	Recipient     string `json:"recipient" gorm:"not null"` // email, phone number, etc.
	CCRecipients  string `json:"cc_recipients" gorm:"type:text"` // comma-separated
	BCCRecipients string `json:"bcc_recipients" gorm:"type:text"` // comma-separated
	
	// Delivery tracking
	SentAt       *time.Time `json:"sent_at" gorm:""`
	DeliveredAt  *time.Time `json:"delivered_at" gorm:""`
	FailedAt     *time.Time `json:"failed_at" gorm:""`
	ReadAt       *time.Time `json:"read_at" gorm:""`
	
	// Retry mechanism
	RetryCount   int        `json:"retry_count" gorm:"default:0"`
	MaxRetries   int        `json:"max_retries" gorm:"default:3"`
	NextRetryAt  *time.Time `json:"next_retry_at" gorm:""`
	
	// Error tracking
	ErrorMessage string `json:"error_message" gorm:"type:text"`
	
	// External IDs (for email services, SMS providers, etc.)
	ExternalID string `json:"external_id" gorm:""`
	
	// Priority and scheduling
	Priority   int        `json:"priority" gorm:"default:0"` // Higher number = higher priority
	ScheduleAt *time.Time `json:"schedule_at" gorm:""`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// EmailVerification represents email verification tokens
type EmailVerification struct {
	ID     uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	Email  string    `json:"email" gorm:"not null;index"`
	
	Token     string    `json:"token" gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	IsUsed    bool      `json:"is_used" gorm:"not null;default:false"`
	UsedAt    *time.Time `json:"used_at" gorm:""`
	
	// Verification attempts
	VerificationAttempts int        `json:"verification_attempts" gorm:"default:0"`
	LastAttemptAt        *time.Time `json:"last_attempt_at" gorm:""`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// PasswordReset represents password reset tokens
type PasswordReset struct {
	ID     uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	Email  string    `json:"email" gorm:"not null;index"`
	
	Token     string    `json:"token" gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	IsUsed    bool      `json:"is_used" gorm:"not null;default:false"`
	UsedAt    *time.Time `json:"used_at" gorm:""`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID     uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	
	// Email preferences
	EmailEnabled              bool `json:"email_enabled" gorm:"not null;default:true"`
	EmailBookingNotifications bool `json:"email_booking_notifications" gorm:"not null;default:true"`
	EmailPaymentNotifications bool `json:"email_payment_notifications" gorm:"not null;default:true"`
	EmailMarketingNotifications bool `json:"email_marketing_notifications" gorm:"not null;default:false"`
	EmailTodoNotifications    bool `json:"email_todo_notifications" gorm:"not null;default:true"`
	EmailReminderNotifications bool `json:"email_reminder_notifications" gorm:"not null;default:true"`
	
	// SMS preferences
	SMSEnabled              bool `json:"sms_enabled" gorm:"not null;default:false"`
	SMSBookingNotifications bool `json:"sms_booking_notifications" gorm:"not null;default:false"`
	SMSReminderNotifications bool `json:"sms_reminder_notifications" gorm:"not null;default:false"`
	
	// In-app preferences
	InAppEnabled              bool `json:"in_app_enabled" gorm:"not null;default:true"`
	InAppBookingNotifications bool `json:"in_app_booking_notifications" gorm:"not null;default:true"`
	InAppTodoNotifications    bool `json:"in_app_todo_notifications" gorm:"not null;default:true"`
	
	// Push preferences
	PushEnabled              bool `json:"push_enabled" gorm:"not null;default:false"`
	PushBookingNotifications bool `json:"push_booking_notifications" gorm:"not null;default:false"`
	PushReminderNotifications bool `json:"push_reminder_notifications" gorm:"not null;default:false"`
	
	// Timing preferences
	QuietHoursEnabled bool      `json:"quiet_hours_enabled" gorm:"not null;default:false"`
	QuietHoursStart   time.Time `json:"quiet_hours_start" gorm:""`
	QuietHoursEnd     time.Time `json:"quiet_hours_end" gorm:""`
	Timezone          string    `json:"timezone" gorm:"default:'Europe/Berlin'"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// ContactForm represents contact form submissions
type ContactForm struct {
	ID uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	
	// Contact information
	Name    string `json:"name" gorm:"not null" validate:"required"`
	Email   string `json:"email" gorm:"not null" validate:"required,email"`
	Phone   string `json:"phone" gorm:""`
	Subject string `json:"subject" gorm:"not null" validate:"required"`
	Message string `json:"message" gorm:"type:text;not null" validate:"required"`
	
	// Additional context
	Source         string `json:"source" gorm:"default:'website'"` // website, landing_page, etc.
	URL            string `json:"url" gorm:""`  // page where form was submitted
	UserAgent      string `json:"user_agent" gorm:"type:text"`
	IPAddress      string `json:"ip_address" gorm:""`
	
	// UTM tracking
	UtmSource   string `json:"utm_source" gorm:""`
	UtmMedium   string `json:"utm_medium" gorm:""`
	UtmCampaign string `json:"utm_campaign" gorm:""`
	UtmTerm     string `json:"utm_term" gorm:""`
	UtmContent  string `json:"utm_content" gorm:""`
	
	// Processing status
	IsProcessed   bool       `json:"is_processed" gorm:"not null;default:false"`
	ProcessedAt   *time.Time `json:"processed_at" gorm:""`
	ProcessedBy   *uuid.UUID `json:"processed_by" gorm:"type:char(36);index"`
	LeadCreated   bool       `json:"lead_created" gorm:"not null;default:false"`
	LeadID        *uuid.UUID `json:"lead_id" gorm:"type:char(36);index"`
	
	// Response tracking
	IsReplied   bool       `json:"is_replied" gorm:"not null;default:false"`
	RepliedAt   *time.Time `json:"replied_at" gorm:""`
	RepliedBy   *uuid.UUID `json:"replied_by" gorm:"type:char(36);index"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Processor *User `json:"processor,omitempty" gorm:"foreignKey:ProcessedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Responder *User `json:"responder,omitempty" gorm:"foreignKey:RepliedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Lead      *Lead `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// Response DTOs
type NotificationResponse struct {
	ID           uuid.UUID          `json:"id"`
	Type         NotificationType   `json:"type"`
	Status       NotificationStatus `json:"status"`
	Title        string             `json:"title"`
	Message      string             `json:"message"`
	Recipient    string             `json:"recipient"`
	SentAt       *time.Time         `json:"sent_at"`
	DeliveredAt  *time.Time         `json:"delivered_at"`
	ReadAt       *time.Time         `json:"read_at"`
	FailedAt     *time.Time         `json:"failed_at"`
	ErrorMessage string             `json:"error_message"`
	RetryCount   int                `json:"retry_count"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type ContactFormResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Subject     string     `json:"subject"`
	Message     string     `json:"message"`
	Source      string     `json:"source"`
	IsProcessed bool       `json:"is_processed"`
	ProcessedAt *time.Time `json:"processed_at"`
	LeadCreated bool       `json:"lead_created"`
	IsReplied   bool       `json:"is_replied"`
	RepliedAt   *time.Time `json:"replied_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Request DTOs
type CreateNotificationRequest struct {
	UserID       uuid.UUID        `json:"user_id" validate:"required"`
	Type         NotificationType `json:"type" validate:"required,oneof=email sms in_app push"`
	Title        string           `json:"title" validate:"required"`
	Message      string           `json:"message" validate:"required"`
	Recipient    string           `json:"recipient" validate:"required"`
	Template     string           `json:"template"`
	TemplateData map[string]interface{} `json:"template_data"`
	ScheduleAt   *time.Time       `json:"schedule_at"`
	Priority     int              `json:"priority"`
}

type CreateContactFormRequest struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone"`
	Subject string `json:"subject" validate:"required"`
	Message string `json:"message" validate:"required"`
	Source  string `json:"source"`
	URL     string `json:"url"`
	
	// UTM parameters
	UtmSource   string `json:"utm_source"`
	UtmMedium   string `json:"utm_medium"`
	UtmCampaign string `json:"utm_campaign"`
	UtmTerm     string `json:"utm_term"`
	UtmContent  string `json:"utm_content"`
}

type UpdateNotificationPreferencesRequest struct {
	EmailEnabled              *bool `json:"email_enabled"`
	EmailBookingNotifications *bool `json:"email_booking_notifications"`
	EmailPaymentNotifications *bool `json:"email_payment_notifications"`
	EmailMarketingNotifications *bool `json:"email_marketing_notifications"`
	EmailTodoNotifications    *bool `json:"email_todo_notifications"`
	EmailReminderNotifications *bool `json:"email_reminder_notifications"`
	SMSEnabled                *bool `json:"sms_enabled"`
	SMSBookingNotifications   *bool `json:"sms_booking_notifications"`
	SMSReminderNotifications  *bool `json:"sms_reminder_notifications"`
	InAppEnabled              *bool `json:"in_app_enabled"`
	InAppBookingNotifications *bool `json:"in_app_booking_notifications"`
	InAppTodoNotifications    *bool `json:"in_app_todo_notifications"`
	PushEnabled               *bool `json:"push_enabled"`
	PushBookingNotifications  *bool `json:"push_booking_notifications"`
	PushReminderNotifications *bool `json:"push_reminder_notifications"`
	QuietHoursEnabled         *bool `json:"quiet_hours_enabled"`
	Timezone                  *string `json:"timezone"`
}

// BeforeCreate hooks
func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

func (ev *EmailVerification) BeforeCreate(tx *gorm.DB) error {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	return nil
}

func (pr *PasswordReset) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}

func (np *NotificationPreference) BeforeCreate(tx *gorm.DB) error {
	if np.ID == uuid.Nil {
		np.ID = uuid.New()
	}
	return nil
}

func (cf *ContactForm) BeforeCreate(tx *gorm.DB) error {
	if cf.ID == uuid.Nil {
		cf.ID = uuid.New()
	}
	return nil
}

// Helper methods
func (n *Notification) ToResponse() NotificationResponse {
	return NotificationResponse{
		ID:           n.ID,
		Type:         n.Type,
		Status:       n.Status,
		Title:        n.Title,
		Message:      n.Message,
		Recipient:    n.Recipient,
		SentAt:       n.SentAt,
		DeliveredAt:  n.DeliveredAt,
		ReadAt:       n.ReadAt,
		FailedAt:     n.FailedAt,
		ErrorMessage: n.ErrorMessage,
		RetryCount:   n.RetryCount,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

func (cf *ContactForm) ToResponse() ContactFormResponse {
	return ContactFormResponse{
		ID:          cf.ID,
		Name:        cf.Name,
		Email:       cf.Email,
		Phone:       cf.Phone,
		Subject:     cf.Subject,
		Message:     cf.Message,
		Source:      cf.Source,
		IsProcessed: cf.IsProcessed,
		ProcessedAt: cf.ProcessedAt,
		LeadCreated: cf.LeadCreated,
		IsReplied:   cf.IsReplied,
		RepliedAt:   cf.RepliedAt,
		CreatedAt:   cf.CreatedAt,
		UpdatedAt:   cf.UpdatedAt,
	}
}

func (n *Notification) CanRetry() bool {
	return n.Status == NotificationStatusFailed && n.RetryCount < n.MaxRetries
}

func (n *Notification) ShouldRetryNow() bool {
	if !n.CanRetry() {
		return false
	}
	if n.NextRetryAt == nil {
		return true
	}
	return time.Now().After(*n.NextRetryAt)
}

func (ev *EmailVerification) IsExpired() bool {
	return time.Now().After(ev.ExpiresAt)
}

func (pr *PasswordReset) IsExpired() bool {
	return time.Now().After(pr.ExpiresAt)
}

func (n *Notification) MarkAsSent() {
	n.Status = NotificationStatusSent
	now := time.Now()
	n.SentAt = &now
}

func (n *Notification) MarkAsDelivered() {
	n.Status = NotificationStatusDelivered
	now := time.Now()
	n.DeliveredAt = &now
}

func (n *Notification) MarkAsFailed(errorMessage string) {
	n.Status = NotificationStatusFailed
	n.ErrorMessage = errorMessage
	now := time.Now()
	n.FailedAt = &now
	n.RetryCount++
	
	// Schedule next retry (exponential backoff)
	if n.CanRetry() {
		nextRetry := time.Now().Add(time.Duration(n.RetryCount*n.RetryCount) * time.Minute)
		n.NextRetryAt = &nextRetry
		n.Status = NotificationStatusRetrying
	}
}

func (ev *EmailVerification) MarkAsUsed() {
	ev.IsUsed = true
	now := time.Now()
	ev.UsedAt = &now
}

func (pr *PasswordReset) MarkAsUsed() {
	pr.IsUsed = true
	now := time.Now()
	pr.UsedAt = &now
}