package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type PaymentMethod string

const (
	PaymentMethodStripe PaymentMethod = "stripe"
	PaymentMethodBank   PaymentMethod = "bank_transfer"
	PaymentMethodCash   PaymentMethod = "cash"
)

// Payment represents a payment transaction
type Payment struct {
	ID     uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	LeadID uuid.UUID `json:"lead_id" gorm:"type:char(36);not null;index"`
	UserID uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`

	// Payment information
	Amount      float64       `json:"amount" gorm:"not null"`
	Currency    string        `json:"currency" gorm:"not null;default:'EUR'"`
	Status      PaymentStatus `json:"status" gorm:"not null;default:'pending'"`
	Method      PaymentMethod `json:"method" gorm:"not null;default:'stripe'"`
	Description string        `json:"description" gorm:"type:text"`

	// Stripe specific fields
	StripeSessionID     string `json:"stripe_session_id" gorm:"uniqueIndex"`
	StripePaymentIntent string `json:"stripe_payment_intent" gorm:""`
	StripeCustomerID    string `json:"stripe_customer_id" gorm:""`
	StripeChargeID      string `json:"stripe_charge_id" gorm:""`

	// Payment details
	PaymentMethodDetails string `json:"payment_method_details" gorm:"type:text"`
	ReceiptURL           string `json:"receipt_url" gorm:""`

	// Billing information
	BillingName    string `json:"billing_name" gorm:""`
	BillingEmail   string `json:"billing_email" gorm:""`
	BillingAddress string `json:"billing_address" gorm:"type:text"`

	// Timestamps
	PaidAt     *time.Time `json:"paid_at" gorm:""`
	FailedAt   *time.Time `json:"failed_at" gorm:""`
	RefundedAt *time.Time `json:"refunded_at" gorm:""`

	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Failure information
	FailureCode    string `json:"failure_code" gorm:""`
	FailureMessage string `json:"failure_message" gorm:"type:text"`

	// Refund information
	RefundAmount float64 `json:"refund_amount" gorm:"default:0"`
	RefundReason string  `json:"refund_reason" gorm:"type:text"`

	// Relationships
	Lead Lead `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// PaymentResponse represents the payment data returned in API responses
type PaymentResponse struct {
	ID                    uuid.UUID     `json:"id"`
	LeadID                uuid.UUID     `json:"lead_id"`
	UserID                uuid.UUID     `json:"user_id"`
	Amount                float64       `json:"amount"`
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	Method                PaymentMethod `json:"method"`
	Description           string        `json:"description"`
	BillingName           string        `json:"billing_name"`
	BillingEmail          string        `json:"billing_email"`
	ReceiptURL            string        `json:"receipt_url"`
	PaidAt                *time.Time    `json:"paid_at"`
	FailedAt              *time.Time    `json:"failed_at"`
	RefundedAt            *time.Time    `json:"refunded_at"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
	FailureCode           string        `json:"failure_code"`
	FailureMessage        string        `json:"failure_message"`
	RefundAmount          float64       `json:"refund_amount"`
	RefundReason          string        `json:"refund_reason"`
	FormattedAmount       string        `json:"formatted_amount"`
	FormattedRefundAmount string        `json:"formatted_refund_amount"`
}

// CreatePaymentRequest represents the request body for creating a payment
type CreatePaymentRequest struct {
	LeadID      uuid.UUID     `json:"lead_id" validate:"required"`
	Amount      float64       `json:"amount" validate:"required,gt=0"`
	Currency    string        `json:"currency" validate:"omitempty,len=3"`
	Description string        `json:"description"`
	Method      PaymentMethod `json:"method" validate:"omitempty,oneof=stripe bank_transfer cash"`
}

// StripeCheckoutRequest represents the request for creating Stripe checkout session
type StripeCheckoutRequest struct {
	LeadID      uuid.UUID `json:"lead_id" validate:"required"`
	Amount      float64   `json:"amount" validate:"required,gt=0"`
	Description string    `json:"description"`
	SuccessURL  string    `json:"success_url"`
	CancelURL   string    `json:"cancel_url"`
}

// StripeCheckoutResponse represents the response for Stripe checkout session
type StripeCheckoutResponse struct {
	SessionID   string    `json:"session_id"`
	CheckoutURL string    `json:"checkout_url"`
	PaymentID   uuid.UUID `json:"payment_id"`
}

// RefundPaymentRequest represents the request for refunding a payment
type RefundPaymentRequest struct {
	Amount float64 `json:"amount" validate:"omitempty,gt=0"`
	Reason string  `json:"reason" validate:"required"`
}

// BeforeCreate is a GORM hook that runs before creating a payment
func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Currency == "" {
		p.Currency = "EUR"
	}
	return nil
}

// ToResponse converts a Payment to PaymentResponse
func (p *Payment) ToResponse() PaymentResponse {
	return PaymentResponse{
		ID:                    p.ID,
		LeadID:                p.LeadID,
		UserID:                p.UserID,
		Amount:                p.Amount,
		Currency:              p.Currency,
		Status:                p.Status,
		Method:                p.Method,
		Description:           p.Description,
		BillingName:           p.BillingName,
		BillingEmail:          p.BillingEmail,
		ReceiptURL:            p.ReceiptURL,
		PaidAt:                p.PaidAt,
		FailedAt:              p.FailedAt,
		RefundedAt:            p.RefundedAt,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
		FailureCode:           p.FailureCode,
		FailureMessage:        p.FailureMessage,
		RefundAmount:          p.RefundAmount,
		RefundReason:          p.RefundReason,
		FormattedAmount:       p.FormatAmount(),
		FormattedRefundAmount: p.FormatRefundAmount(),
	}
}

// FormatAmount returns the formatted amount with currency
func (p *Payment) FormatAmount() string {
	return fmt.Sprintf("%.2f %s", p.Amount, p.Currency)
}

// FormatRefundAmount returns the formatted refund amount with currency
func (p *Payment) FormatRefundAmount() string {
	if p.RefundAmount > 0 {
		return fmt.Sprintf("%.2f %s", p.RefundAmount, p.Currency)
	}
	return ""
}

// IsPaid checks if the payment is paid
func (p *Payment) IsPaid() bool {
	return p.Status == PaymentStatusSucceeded
}

// IsFailed checks if the payment failed
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

// IsRefunded checks if the payment is refunded
func (p *Payment) IsRefunded() bool {
	return p.Status == PaymentStatusRefunded || p.RefundAmount > 0
}

// IsPending checks if the payment is pending
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending || p.Status == PaymentStatusProcessing
}

// CanBeRefunded checks if the payment can be refunded
func (p *Payment) CanBeRefunded() bool {
	return p.IsPaid() && !p.IsRefunded()
}

// GetRemainingRefundAmount returns the remaining amount that can be refunded
func (p *Payment) GetRemainingRefundAmount() float64 {
	if !p.CanBeRefunded() {
		return 0
	}
	return p.Amount - p.RefundAmount
}

// MarkAsPaid marks the payment as paid
func (p *Payment) MarkAsPaid() {
	p.Status = PaymentStatusSucceeded
	now := time.Now()
	p.PaidAt = &now
}

// MarkAsFailed marks the payment as failed
func (p *Payment) MarkAsFailed(code, message string) {
	p.Status = PaymentStatusFailed
	p.FailureCode = code
	p.FailureMessage = message
	now := time.Now()
	p.FailedAt = &now
}

// MarkAsRefunded marks the payment as refunded
func (p *Payment) MarkAsRefunded(amount float64, reason string) {
	if amount >= p.Amount {
		p.Status = PaymentStatusRefunded
	}
	p.RefundAmount += amount
	p.RefundReason = reason
	now := time.Now()
	p.RefundedAt = &now
}

// GetDisplayName returns a human-readable display name for the payment status
func (ps PaymentStatus) GetDisplayName() string {
	switch ps {
	case PaymentStatusPending:
		return "Ausstehend"
	case PaymentStatusProcessing:
		return "In Bearbeitung"
	case PaymentStatusSucceeded:
		return "Erfolgreich"
	case PaymentStatusFailed:
		return "Fehlgeschlagen"
	case PaymentStatusCanceled:
		return "Abgebrochen"
	case PaymentStatusRefunded:
		return "Rückerstattet"
	default:
		return "Unbekannt"
	}
}

// GetDisplayName returns a human-readable display name for the payment method
func (pm PaymentMethod) GetDisplayName() string {
	switch pm {
	case PaymentMethodStripe:
		return "Kreditkarte/Online"
	case PaymentMethodBank:
		return "Banküberweisung"
	case PaymentMethodCash:
		return "Bar"
	default:
		return "Unbekannt"
	}
}

// GetColorClass returns a CSS color class for the payment status (for frontend usage)
func (ps PaymentStatus) GetColorClass() string {
	switch ps {
	case PaymentStatusPending:
		return "warning"
	case PaymentStatusProcessing:
		return "info"
	case PaymentStatusSucceeded:
		return "success"
	case PaymentStatusFailed:
		return "danger"
	case PaymentStatusCanceled:
		return "secondary"
	case PaymentStatusRefunded:
		return "info"
	default:
		return "secondary"
	}
}
