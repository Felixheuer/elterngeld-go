package models

// PackageTypeService represents a service-based package type
type PackageTypeService string

const (
	PackageTypeServiceConsultation PackageTypeService = "consultation"
	PackageTypeServiceApplication  PackageTypeService = "application"
	PackageTypeServiceReview      PackageTypeService = "review"
)

// PackageTypeAddOn represents an add-on package type
type PackageTypeAddOn string

const (
	PackageTypeAddOnDocument   PackageTypeAddOn = "document"
	PackageTypeAddOnTranslation PackageTypeAddOn = "translation"
	PackageTypeAddOnSupport    PackageTypeAddOn = "support"
)

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PackageType string

const (
	PackageTypeBasic    PackageType = "basic"
	PackageTypePremium  PackageType = "premium"
	PackageTypeComplete PackageType = "complete"
)

// Package represents service packages that users can book
type Package struct {
	ID          uuid.UUID   `json:"id" gorm:"type:char(36);primary_key"`
	Name        string      `json:"name" gorm:"not null" validate:"required"`
	Description string      `json:"description" gorm:"type:text"`
	Type        PackageType `json:"type" gorm:"not null" validate:"required"`
	
	// Pricing
	Price         float64 `json:"price" gorm:"not null" validate:"required,gte=0"`
	Currency      string  `json:"currency" gorm:"not null;default:'EUR'"`
	IsActive      bool    `json:"is_active" gorm:"not null;default:true"`
	
	// Stripe integration
	StripeProductID string `json:"stripe_product_id" gorm:"uniqueIndex"`
	StripePriceID   string `json:"stripe_price_id" gorm:"uniqueIndex"`
	
	// Package features and settings
	Features           string `json:"features" gorm:"type:text"` // JSON array of features
	RequiresTimeslot   bool   `json:"requires_timeslot" gorm:"not null;default:true"`
	ManualAssignment   bool   `json:"manual_assignment" gorm:"not null;default:false"`
	ConsultationTime   int    `json:"consultation_time" gorm:"default:60"` // in minutes
	HasFreePreTalk     bool   `json:"has_free_pre_talk" gorm:"not null;default:false"`
	PreTalkDuration    int    `json:"pre_talk_duration" gorm:"default:15"` // in minutes
	
	// Display settings
	SortOrder   int    `json:"sort_order" gorm:"default:0"`
	BadgeText   string `json:"badge_text" gorm:""`
	BadgeColor  string `json:"badge_color" gorm:"default:'primary'"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Addons   []Addon   `json:"addons,omitempty" gorm:"many2many:package_addons;"`
	Bookings []Booking `json:"bookings,omitempty" gorm:"foreignKey:PackageID"`
}

// Addon represents additional services that can be added to packages
type Addon struct {
	ID          uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	Name        string    `json:"name" gorm:"not null" validate:"required"`
	Description string    `json:"description" gorm:"type:text"`
	
	// Pricing
	Price    float64 `json:"price" gorm:"not null" validate:"required,gte=0"`
	Currency string  `json:"currency" gorm:"not null;default:'EUR'"`
	IsActive bool    `json:"is_active" gorm:"not null;default:true"`
	
	// Stripe integration
	StripeProductID string `json:"stripe_product_id" gorm:"uniqueIndex"`
	StripePriceID   string `json:"stripe_price_id" gorm:"uniqueIndex"`
	
	// Display settings
	SortOrder int    `json:"sort_order" gorm:"default:0"`
	Category  string `json:"category" gorm:"default:'general'"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Packages []Package `json:"packages,omitempty" gorm:"many2many:package_addons;"`
	Bookings []Booking `json:"bookings,omitempty" gorm:"many2many:booking_addons;"`
}

// PackageAddon represents the junction table for package-addon relationships
type PackageAddon struct {
	PackageID uuid.UUID `json:"package_id" gorm:"type:char(36);primary_key"`
	AddonID   uuid.UUID `json:"addon_id" gorm:"type:char(36);primary_key"`
	IsDefault bool      `json:"is_default" gorm:"not null;default:false"`
	
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	
	// Relationships
	Package Package `json:"package,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Addon   Addon   `json:"addon,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// PackageResponse represents the package data returned in API responses
type PackageResponse struct {
	ID                 uuid.UUID      `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Type               PackageType    `json:"type"`
	Price              float64        `json:"price"`
	Currency           string         `json:"currency"`
	FormattedPrice     string         `json:"formatted_price"`
	IsActive           bool           `json:"is_active"`
	Features           []string       `json:"features"`
	RequiresTimeslot   bool           `json:"requires_timeslot"`
	ManualAssignment   bool           `json:"manual_assignment"`
	ConsultationTime   int            `json:"consultation_time"`
	HasFreePreTalk     bool           `json:"has_free_pre_talk"`
	PreTalkDuration    int            `json:"pre_talk_duration"`
	SortOrder          int            `json:"sort_order"`
	BadgeText          string         `json:"badge_text"`
	BadgeColor         string         `json:"badge_color"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	AvailableAddons    []AddonResponse `json:"available_addons,omitempty"`
}

// AddonResponse represents the addon data returned in API responses
type AddonResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"`
	Currency       string    `json:"currency"`
	FormattedPrice string    `json:"formatted_price"`
	IsActive       bool      `json:"is_active"`
	SortOrder      int       `json:"sort_order"`
	Category       string    `json:"category"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreatePackageRequest represents the request body for creating a package
type CreatePackageRequest struct {
	Name             string      `json:"name" validate:"required"`
	Description      string      `json:"description"`
	Type             PackageType `json:"type" validate:"required,oneof=basic premium complete"`
	Price            float64     `json:"price" validate:"required,gte=0"`
	Features         []string    `json:"features"`
	RequiresTimeslot bool        `json:"requires_timeslot"`
	ManualAssignment bool        `json:"manual_assignment"`
	ConsultationTime int         `json:"consultation_time" validate:"gte=0"`
	HasFreePreTalk   bool        `json:"has_free_pre_talk"`
	PreTalkDuration  int         `json:"pre_talk_duration" validate:"gte=0"`
	BadgeText        string      `json:"badge_text"`
	BadgeColor       string      `json:"badge_color"`
	SortOrder        int         `json:"sort_order"`
}

// UpdatePackageRequest represents the request body for updating a package
type UpdatePackageRequest struct {
	Name             *string      `json:"name"`
	Description      *string      `json:"description"`
	Type             *PackageType `json:"type" validate:"omitempty,oneof=basic premium complete"`
	Price            *float64     `json:"price" validate:"omitempty,gte=0"`
	Features         []string     `json:"features"`
	RequiresTimeslot *bool        `json:"requires_timeslot"`
	ManualAssignment *bool        `json:"manual_assignment"`
	ConsultationTime *int         `json:"consultation_time" validate:"omitempty,gte=0"`
	HasFreePreTalk   *bool        `json:"has_free_pre_talk"`
	PreTalkDuration  *int         `json:"pre_talk_duration" validate:"omitempty,gte=0"`
	BadgeText        *string      `json:"badge_text"`
	BadgeColor       *string      `json:"badge_color"`
	SortOrder        *int         `json:"sort_order"`
	IsActive         *bool        `json:"is_active"`
}

// CreateAddonRequest represents the request body for creating an addon
type CreateAddonRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" validate:"required,gte=0"`
	Category    string  `json:"category"`
	SortOrder   int     `json:"sort_order"`
}

// UpdateAddonRequest represents the request body for updating an addon
type UpdateAddonRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price" validate:"omitempty,gte=0"`
	Category    *string  `json:"category"`
	SortOrder   *int     `json:"sort_order"`
	IsActive    *bool    `json:"is_active"`
}

// BeforeCreate is a GORM hook that runs before creating a package
func (p *Package) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Currency == "" {
		p.Currency = "EUR"
	}
	return nil
}

// BeforeCreate is a GORM hook that runs before creating an addon
func (a *Addon) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Currency == "" {
		a.Currency = "EUR"
	}
	return nil
}

// ToResponse converts a Package to PackageResponse
func (p *Package) ToResponse() PackageResponse {
	response := PackageResponse{
		ID:               p.ID,
		Name:             p.Name,
		Description:      p.Description,
		Type:             p.Type,
		Price:            p.Price,
		Currency:         p.Currency,
		FormattedPrice:   p.FormatPrice(),
		IsActive:         p.IsActive,
		Features:         p.GetFeaturesArray(),
		RequiresTimeslot: p.RequiresTimeslot,
		ManualAssignment: p.ManualAssignment,
		ConsultationTime: p.ConsultationTime,
		HasFreePreTalk:   p.HasFreePreTalk,
		PreTalkDuration:  p.PreTalkDuration,
		SortOrder:        p.SortOrder,
		BadgeText:        p.BadgeText,
		BadgeColor:       p.BadgeColor,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
	
	// Convert addons
	for _, addon := range p.Addons {
		response.AvailableAddons = append(response.AvailableAddons, addon.ToResponse())
	}
	
	return response
}

// ToResponse converts an Addon to AddonResponse
func (a *Addon) ToResponse() AddonResponse {
	return AddonResponse{
		ID:             a.ID,
		Name:           a.Name,
		Description:    a.Description,
		Price:          a.Price,
		Currency:       a.Currency,
		FormattedPrice: a.FormatPrice(),
		IsActive:       a.IsActive,
		SortOrder:      a.SortOrder,
		Category:       a.Category,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// FormatPrice returns the formatted price with currency
func (p *Package) FormatPrice() string {
	return formatCurrency(p.Price, p.Currency)
}

// FormatPrice returns the formatted price with currency
func (a *Addon) FormatPrice() string {
	return formatCurrency(a.Price, a.Currency)
}

// GetFeaturesArray parses the features JSON string into a slice
func (p *Package) GetFeaturesArray() []string {
	// This would normally parse JSON, but for simplicity we'll return empty slice
	// In real implementation, you'd unmarshal the JSON string
	return []string{}
}

// Helper function to format currency (could be moved to utils)
func formatCurrency(amount float64, currency string) string {
	switch currency {
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}

// GetDisplayName returns a human-readable display name for the package type
func (pt PackageType) GetDisplayName() string {
	switch pt {
	case PackageTypeBasic:
		return "Basis"
	case PackageTypePremium:
		return "Premium"
	case PackageTypeComplete:
		return "Komplett"
	default:
		return "Unbekannt"
	}
}