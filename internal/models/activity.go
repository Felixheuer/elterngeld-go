package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityType string

const (
	ActivityTypeLeadCreated        ActivityType = "lead_created"
	ActivityTypeLeadUpdated        ActivityType = "lead_updated"
	ActivityTypeLeadStatusChanged  ActivityType = "lead_status_changed"
	ActivityTypeLeadAssigned       ActivityType = "lead_assigned"
	ActivityTypeCommentAdded       ActivityType = "comment_added"
	ActivityTypeDocumentUploaded   ActivityType = "document_uploaded"
	ActivityTypeDocumentDeleted    ActivityType = "document_deleted"
	ActivityTypePaymentCreated     ActivityType = "payment_created"
	ActivityTypePaymentCompleted   ActivityType = "payment_completed"
	ActivityTypePaymentFailed      ActivityType = "payment_failed"
	ActivityTypeUserRegistered     ActivityType = "user_registered"
	ActivityTypeUserLogin          ActivityType = "user_login"
	ActivityTypeUserLogout         ActivityType = "user_logout"
	ActivityTypePasswordChanged    ActivityType = "password_changed"
	ActivityTypeEmailSent          ActivityType = "email_sent"
	ActivityTypeSystem             ActivityType = "system"
)

// Activity represents an activity/event in the system
type Activity struct {
	ID       uuid.UUID    `json:"id" gorm:"type:char(36);primary_key"`
	UserID   *uuid.UUID   `json:"user_id" gorm:"type:char(36);index"`
	LeadID   *uuid.UUID   `json:"lead_id" gorm:"type:char(36);index"`
	
	// Activity information
	Type        ActivityType `json:"type" gorm:"not null;index" validate:"required"`
	Title       string       `json:"title" gorm:"not null" validate:"required"`
	Description string       `json:"description" gorm:"type:text"`
	
	// Metadata as JSON
	Metadata    json.RawMessage `json:"metadata" gorm:"type:jsonb"`
	
	// Request information
	IPAddress string `json:"ip_address" gorm:""`
	UserAgent string `json:"user_agent" gorm:""`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" gorm:"not null;index"`
	
	// Relationships
	User *User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Lead *Lead `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// ActivityResponse represents the activity data returned in API responses
type ActivityResponse struct {
	ID          uuid.UUID       `json:"id"`
	UserID      *uuid.UUID      `json:"user_id"`
	LeadID      *uuid.UUID      `json:"lead_id"`
	Type        ActivityType    `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	IPAddress   string          `json:"ip_address"`
	CreatedAt   time.Time       `json:"created_at"`
	User        *UserResponse   `json:"user,omitempty"`
}

// ActivityMetadata represents common metadata structure
type ActivityMetadata struct {
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Field       string      `json:"field,omitempty"`
	EntityID    string      `json:"entity_id,omitempty"`
	EntityType  string      `json:"entity_type,omitempty"`
	ExtraData   interface{} `json:"extra_data,omitempty"`
}

// BeforeCreate is a GORM hook that runs before creating an activity
func (a *Activity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// ToResponse converts an Activity to ActivityResponse
func (a *Activity) ToResponse() ActivityResponse {
	response := ActivityResponse{
		ID:          a.ID,
		UserID:      a.UserID,
		LeadID:      a.LeadID,
		Type:        a.Type,
		Title:       a.Title,
		Description: a.Description,
		Metadata:    a.Metadata,
		IPAddress:   a.IPAddress,
		CreatedAt:   a.CreatedAt,
	}
	
	if a.User != nil && a.User.ID != uuid.Nil {
		userResponse := a.User.ToResponse()
		response.User = &userResponse
	}
	
	return response
}

// SetMetadata sets the metadata field with proper JSON encoding
func (a *Activity) SetMetadata(metadata interface{}) error {
	if metadata == nil {
		a.Metadata = nil
		return nil
	}
	
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	
	a.Metadata = data
	return nil
}

// GetMetadata retrieves and unmarshals the metadata
func (a *Activity) GetMetadata(target interface{}) error {
	if a.Metadata == nil {
		return nil
	}
	
	return json.Unmarshal(a.Metadata, target)
}

// ActivityBuilder provides a fluent interface for creating activities
type ActivityBuilder struct {
	activity *Activity
}

// NewActivityBuilder creates a new activity builder
func NewActivityBuilder() *ActivityBuilder {
	return &ActivityBuilder{
		activity: &Activity{
			ID: uuid.New(),
		},
	}
}

// WithType sets the activity type
func (ab *ActivityBuilder) WithType(activityType ActivityType) *ActivityBuilder {
	ab.activity.Type = activityType
	return ab
}

// WithTitle sets the activity title
func (ab *ActivityBuilder) WithTitle(title string) *ActivityBuilder {
	ab.activity.Title = title
	return ab
}

// WithDescription sets the activity description
func (ab *ActivityBuilder) WithDescription(description string) *ActivityBuilder {
	ab.activity.Description = description
	return ab
}

// WithUser sets the user ID
func (ab *ActivityBuilder) WithUser(userID uuid.UUID) *ActivityBuilder {
	ab.activity.UserID = &userID
	return ab
}

// WithLead sets the lead ID
func (ab *ActivityBuilder) WithLead(leadID uuid.UUID) *ActivityBuilder {
	ab.activity.LeadID = &leadID
	return ab
}

// WithMetadata sets the metadata
func (ab *ActivityBuilder) WithMetadata(metadata interface{}) *ActivityBuilder {
	ab.activity.SetMetadata(metadata)
	return ab
}

// WithIPAddress sets the IP address
func (ab *ActivityBuilder) WithIPAddress(ipAddress string) *ActivityBuilder {
	ab.activity.IPAddress = ipAddress
	return ab
}

// WithUserAgent sets the user agent
func (ab *ActivityBuilder) WithUserAgent(userAgent string) *ActivityBuilder {
	ab.activity.UserAgent = userAgent
	return ab
}

// Build returns the built activity
func (ab *ActivityBuilder) Build() *Activity {
	return ab.activity
}

// GetDisplayName returns a human-readable display name for the activity type
func (at ActivityType) GetDisplayName() string {
	switch at {
	case ActivityTypeLeadCreated:
		return "Lead erstellt"
	case ActivityTypeLeadUpdated:
		return "Lead aktualisiert"
	case ActivityTypeLeadStatusChanged:
		return "Lead-Status geändert"
	case ActivityTypeLeadAssigned:
		return "Lead zugewiesen"
	case ActivityTypeCommentAdded:
		return "Kommentar hinzugefügt"
	case ActivityTypeDocumentUploaded:
		return "Dokument hochgeladen"
	case ActivityTypeDocumentDeleted:
		return "Dokument gelöscht"
	case ActivityTypePaymentCreated:
		return "Zahlung erstellt"
	case ActivityTypePaymentCompleted:
		return "Zahlung abgeschlossen"
	case ActivityTypePaymentFailed:
		return "Zahlung fehlgeschlagen"
	case ActivityTypeUserRegistered:
		return "Benutzer registriert"
	case ActivityTypeUserLogin:
		return "Benutzer angemeldet"
	case ActivityTypeUserLogout:
		return "Benutzer abgemeldet"
	case ActivityTypePasswordChanged:
		return "Passwort geändert"
	case ActivityTypeEmailSent:
		return "E-Mail gesendet"
	case ActivityTypeSystem:
		return "System-Aktivität"
	default:
		return "Unbekannte Aktivität"
	}
}

// GetIconName returns an icon name for the activity type (for frontend usage)
func (at ActivityType) GetIconName() string {
	switch at {
	case ActivityTypeLeadCreated:
		return "plus-circle"
	case ActivityTypeLeadUpdated:
		return "edit"
	case ActivityTypeLeadStatusChanged:
		return "refresh"
	case ActivityTypeLeadAssigned:
		return "user-plus"
	case ActivityTypeCommentAdded:
		return "message-circle"
	case ActivityTypeDocumentUploaded:
		return "upload"
	case ActivityTypeDocumentDeleted:
		return "trash-2"
	case ActivityTypePaymentCreated:
		return "credit-card"
	case ActivityTypePaymentCompleted:
		return "check-circle"
	case ActivityTypePaymentFailed:
		return "x-circle"
	case ActivityTypeUserRegistered:
		return "user-plus"
	case ActivityTypeUserLogin:
		return "log-in"
	case ActivityTypeUserLogout:
		return "log-out"
	case ActivityTypePasswordChanged:
		return "lock"
	case ActivityTypeEmailSent:
		return "mail"
	case ActivityTypeSystem:
		return "settings"
	default:
		return "help-circle"
	}
}

// Helper functions for creating common activities

// CreateLeadCreatedActivity creates an activity for lead creation
func CreateLeadCreatedActivity(userID, leadID uuid.UUID, leadTitle string) *Activity {
	return NewActivityBuilder().
		WithType(ActivityTypeLeadCreated).
		WithTitle("Neuer Lead erstellt").
		WithDescription("Lead '" + leadTitle + "' wurde erstellt").
		WithUser(userID).
		WithLead(leadID).
		Build()
}

// CreateLeadStatusChangedActivity creates an activity for lead status change
func CreateLeadStatusChangedActivity(userID, leadID uuid.UUID, oldStatus, newStatus LeadStatus) *Activity {
	metadata := ActivityMetadata{
		OldValue: string(oldStatus),
		NewValue: string(newStatus),
		Field:    "status",
	}
	
	return NewActivityBuilder().
		WithType(ActivityTypeLeadStatusChanged).
		WithTitle("Lead-Status geändert").
		WithDescription(fmt.Sprintf("Status von '%s' zu '%s' geändert", oldStatus, newStatus)).
		WithUser(userID).
		WithLead(leadID).
		WithMetadata(metadata).
		Build()
}

// CreateDocumentUploadedActivity creates an activity for document upload
func CreateDocumentUploadedActivity(userID, leadID uuid.UUID, fileName string, documentType DocumentType) *Activity {
	metadata := ActivityMetadata{
		EntityType: "document",
		ExtraData: map[string]interface{}{
			"file_name":     fileName,
			"document_type": documentType,
		},
	}
	
	return NewActivityBuilder().
		WithType(ActivityTypeDocumentUploaded).
		WithTitle("Dokument hochgeladen").
		WithDescription("Dokument '" + fileName + "' wurde hochgeladen").
		WithUser(userID).
		WithLead(leadID).
		WithMetadata(metadata).
		Build()
}

// CreatePaymentCompletedActivity creates an activity for payment completion
func CreatePaymentCompletedActivity(userID, leadID uuid.UUID, amount float64, currency string) *Activity {
	metadata := ActivityMetadata{
		ExtraData: map[string]interface{}{
			"amount":   amount,
			"currency": currency,
		},
	}
	
	return NewActivityBuilder().
		WithType(ActivityTypePaymentCompleted).
		WithTitle("Zahlung abgeschlossen").
		WithDescription(fmt.Sprintf("Zahlung über %.2f %s wurde erfolgreich abgeschlossen", amount, currency)).
		WithUser(userID).
		WithLead(leadID).
		WithMetadata(metadata).
		Build()
}