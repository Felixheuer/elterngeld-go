package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/webhook"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

func NewPaymentHandler(db *gorm.DB, logger *zap.Logger, config *config.Config) *PaymentHandler {
	// Initialize Stripe
	stripe.Key = config.Stripe.SecretKey
	
	return &PaymentHandler{
		db:     db,
		logger: logger,
		config: config,
	}
}

// CreateCheckoutRequest represents the checkout creation request
type CreateCheckoutRequest struct {
	BookingID   uuid.UUID `json:"booking_id" binding:"required"`
	SuccessURL  string    `json:"success_url,omitempty"`
	CancelURL   string    `json:"cancel_url,omitempty"`
}

// RefundRequest represents the refund request
type RefundRequest struct {
	Amount *int64  `json:"amount,omitempty"` // Amount in cents, if nil refund full amount
	Reason string  `json:"reason,omitempty"`
}

// ListPayments handles listing payments for a user
// @Summary List payments
// @Description Get list of payments for current user
// @Tags payments
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/payments [get]
func (h *PaymentHandler) ListPayments(c *gin.Context) {
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
	query.Model(&models.Payment{}).Count(&total)

	// Get payments with preloaded relations
	var payments []models.Payment
	if err := query.Preload("Booking").Preload("Booking.Package").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&payments).Error; err != nil {
		h.logger.Error("Failed to fetch payments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// CreateCheckout handles creating a Stripe checkout session
// @Summary Create Stripe checkout
// @Description Create a Stripe checkout session for a booking
// @Tags payments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateCheckoutRequest true "Checkout data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/payments/checkout [post]
func (h *PaymentHandler) CreateCheckout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Get booking with related data
	var booking models.Booking
	if err := h.db.Where("id = ? AND user_id = ?", req.BookingID, userID).
		Preload("Package").Preload("User").First(&booking).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		} else {
			h.logger.Error("Failed to fetch booking", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booking"})
		}
		return
	}

	// Check if booking already has a successful payment
	var existingPayment models.Payment
	if err := h.db.Where("booking_id = ? AND status = ?", booking.ID, models.PaymentStatusCompleted).
		First(&existingPayment).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Booking already paid"})
		return
	}

	// Get add-ons for line items
	var addOns []models.Package
	h.db.Table("booking_add_ons").
		Select("packages.*, booking_add_ons.price as addon_price").
		Joins("JOIN packages ON packages.id = booking_add_ons.package_id").
		Where("booking_add_ons.booking_id = ?", booking.ID).
		Find(&addOns)

	// Prepare line items for Stripe
	var lineItems []*stripe.CheckoutSessionLineItemParams

	// Main package
	lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
		PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
			Currency: stripe.String(string(stripe.CurrencyEUR)),
			ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
				Name:        stripe.String(booking.Package.Name),
				Description: stripe.String(booking.Package.Description),
			},
			UnitAmount: stripe.Int64(int64(booking.Package.Price * 100)), // Convert to cents
		},
		Quantity: stripe.Int64(1),
	})

	// Add-ons
	for _, addOn := range addOns {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemPriceDataParams{
			Currency: stripe.String(string(stripe.CurrencyEUR)),
			ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
				Name:        stripe.String(addOn.Name + " (Add-On)"),
				Description: stripe.String(addOn.Description),
			},
			UnitAmount: stripe.Int64(int64(addOn.Price * 100)), // Convert to cents
		})
	}

	// Set default URLs if not provided
	successURL := req.SuccessURL
	if successURL == "" {
		successURL = h.config.App.BaseURL + "/payment/success?session_id={CHECKOUT_SESSION_ID}"
	}

	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = h.config.App.BaseURL + "/payment/cancel"
	}

	// Create Stripe checkout session
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems:          lineItems,
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String(successURL),
		CancelURL:          stripe.String(cancelURL),
		CustomerEmail:      stripe.String(booking.User.Email),
		Metadata: map[string]string{
			"booking_id": booking.ID.String(),
			"user_id":    userID.(uuid.UUID).String(),
		},
		ExpiresAt: stripe.Int64(time.Now().Add(24 * time.Hour).Unix()), // 24 hour expiry
	}

	session, err := session.New(params)
	if err != nil {
		h.logger.Error("Failed to create Stripe session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session"})
		return
	}

	// Create payment record
	payment := models.Payment{
		ID:               uuid.New(),
		UserID:           userID.(uuid.UUID),
		BookingID:        &booking.ID,
		StripeSessionID:  &session.ID,
		Status:           models.PaymentStatusPending,
		Amount:           booking.TotalPrice,
		Currency:         booking.Currency,
		PaymentMethod:    models.PaymentMethodCard,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.db.Create(&payment).Error; err != nil {
		h.logger.Error("Failed to create payment record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}

	h.logger.Info("Checkout session created", 
		zap.String("session_id", session.ID),
		zap.String("booking_id", booking.ID.String()))

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": session.URL,
		"session_id":   session.ID,
		"payment_id":   payment.ID,
	})
}

// GetPayment handles getting a specific payment
// @Summary Get payment by ID
// @Description Get payment details
// @Tags payments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} models.Payment
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/payments/{id} [get]
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	paymentID := c.Param("id")

	var payment models.Payment
	query := h.db.Where("id = ?", paymentID).Preload("Booking").Preload("Booking.Package")

	// Non-admin users can only see their own payments
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && userRole != "berater" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		} else {
			h.logger.Error("Failed to fetch payment", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment"})
		}
		return
	}

	c.JSON(http.StatusOK, payment)
}

// RefundPayment handles creating a refund for a payment
// @Summary Refund payment
// @Description Create a refund for a payment (Berater/Admin only)
// @Tags payments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param request body RefundRequest true "Refund data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/payments/{id}/refund [post]
func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	paymentID := c.Param("id")

	var payment models.Payment
	if err := h.db.Where("id = ?", paymentID).Preload("Booking").First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		} else {
			h.logger.Error("Failed to fetch payment", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment"})
		}
		return
	}

	// Check if payment can be refunded
	if payment.Status != models.PaymentStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment is not completed"})
		return
	}

	if payment.StripePaymentIntentID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Stripe payment intent found"})
		return
	}

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Calculate refund amount
	refundAmount := req.Amount
	if refundAmount == nil {
		// Full refund
		refundAmount = stripe.Int64(int64(payment.Amount * 100)) // Convert to cents
	}

	// Create Stripe refund
	refundParams := &stripe.RefundParams{
		PaymentIntent: payment.StripePaymentIntentID,
		Amount:        refundAmount,
	}

	if req.Reason != "" {
		refundParams.Reason = stripe.String(req.Reason)
	}

	stripeRefund, err := refund.New(refundParams)
	if err != nil {
		h.logger.Error("Failed to create Stripe refund", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create refund"})
		return
	}

	// Update payment status
	refundAmountFloat := float64(*refundAmount) / 100
	if refundAmountFloat >= payment.Amount {
		payment.Status = models.PaymentStatusRefunded
	} else {
		payment.Status = models.PaymentStatusPartiallyRefunded
	}

	payment.StripeRefundID = &stripeRefund.ID
	payment.RefundAmount = &refundAmountFloat
	payment.RefundedAt = &time.Time{}
	*payment.RefundedAt = time.Now()
	payment.UpdatedAt = time.Now()

	if err := h.db.Save(&payment).Error; err != nil {
		h.logger.Error("Failed to update payment after refund", zap.Error(err))
		// Note: Refund was successful in Stripe, but we failed to update our DB
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Refund created but failed to update record"})
		return
	}

	h.logger.Info("Payment refunded", 
		zap.String("payment_id", paymentID),
		zap.Float64("amount", refundAmountFloat))

	c.JSON(http.StatusOK, gin.H{
		"message":      "Refund created successfully",
		"refund_id":    stripeRefund.ID,
		"amount":       refundAmountFloat,
		"status":       stripeRefund.Status,
		"payment":      payment,
	})
}

// StripeWebhook handles Stripe webhook events
// @Summary Stripe webhook
// @Description Handle Stripe webhook events
// @Tags webhooks
// @Accept json
// @Produce json
// @Param stripe-signature header string true "Stripe signature"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/webhooks/stripe [post]
func (h *PaymentHandler) StripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(body, c.GetHeader("Stripe-Signature"), h.config.Stripe.WebhookSecret)
	if err != nil {
		h.logger.Error("Failed to verify webhook signature", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	h.logger.Info("Received Stripe webhook", zap.String("type", string(event.Type)))

	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutSessionCompleted(event)
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(event)
	case "payment_intent.payment_failed":
		h.handlePaymentIntentFailed(event)
	case "invoice.payment_succeeded":
		h.handleInvoicePaymentSucceeded(event)
	case "customer.subscription.created":
		h.handleSubscriptionCreated(event)
	default:
		h.logger.Info("Unhandled webhook event type", zap.String("type", string(event.Type)))
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleCheckoutSessionCompleted handles successful checkout sessions
func (h *PaymentHandler) handleCheckoutSessionCompleted(event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		h.logger.Error("Failed to parse checkout session", zap.Error(err))
		return
	}

	// Get booking ID from metadata
	bookingID, exists := session.Metadata["booking_id"]
	if !exists {
		h.logger.Error("No booking_id in session metadata")
		return
	}

	// Update payment record
	var payment models.Payment
	if err := h.db.Where("stripe_session_id = ?", session.ID).First(&payment).Error; err != nil {
		h.logger.Error("Failed to find payment by session ID", zap.String("session_id", session.ID))
		return
	}

	// Update payment status
	payment.Status = models.PaymentStatusCompleted
	payment.StripePaymentIntentID = &session.PaymentIntent.ID
	payment.CompletedAt = &time.Time{}
	*payment.CompletedAt = time.Now()
	payment.UpdatedAt = time.Now()

	if err := h.db.Save(&payment).Error; err != nil {
		h.logger.Error("Failed to update payment", zap.Error(err))
		return
	}

	// Update booking status
	var booking models.Booking
	if err := h.db.Where("id = ?", bookingID).First(&booking).Error; err != nil {
		h.logger.Error("Failed to find booking", zap.String("booking_id", bookingID))
		return
	}

	booking.Status = models.BookingStatusConfirmed
	booking.UpdatedAt = time.Now()

	if err := h.db.Save(&booking).Error; err != nil {
		h.logger.Error("Failed to update booking status", zap.Error(err))
		return
	}

	h.logger.Info("Payment completed successfully", 
		zap.String("payment_id", payment.ID.String()),
		zap.String("booking_id", bookingID))

	// TODO: Send confirmation email
}

// handlePaymentIntentSucceeded handles successful payment intents
func (h *PaymentHandler) handlePaymentIntentSucceeded(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		h.logger.Error("Failed to parse payment intent", zap.Error(err))
		return
	}

	// Update payment record if exists
	var payment models.Payment
	if err := h.db.Where("stripe_payment_intent_id = ?", paymentIntent.ID).First(&payment).Error; err != nil {
		// Payment might not exist in our system yet, that's okay
		return
	}

	payment.Status = models.PaymentStatusCompleted
	payment.UpdatedAt = time.Now()

	if err := h.db.Save(&payment).Error; err != nil {
		h.logger.Error("Failed to update payment", zap.Error(err))
	}
}

// handlePaymentIntentFailed handles failed payment intents
func (h *PaymentHandler) handlePaymentIntentFailed(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		h.logger.Error("Failed to parse payment intent", zap.Error(err))
		return
	}

	// Update payment record if exists
	var payment models.Payment
	if err := h.db.Where("stripe_payment_intent_id = ?", paymentIntent.ID).First(&payment).Error; err != nil {
		return
	}

	payment.Status = models.PaymentStatusFailed
	payment.UpdatedAt = time.Now()

	if err := h.db.Save(&payment).Error; err != nil {
		h.logger.Error("Failed to update payment", zap.Error(err))
	}
}

// handleInvoicePaymentSucceeded handles successful invoice payments (for subscriptions)
func (h *PaymentHandler) handleInvoicePaymentSucceeded(event stripe.Event) {
	// TODO: Implement subscription handling if needed
	h.logger.Info("Invoice payment succeeded", zap.String("event_id", event.ID))
}

// handleSubscriptionCreated handles new subscription creation
func (h *PaymentHandler) handleSubscriptionCreated(event stripe.Event) {
	// TODO: Implement subscription handling if needed
	h.logger.Info("Subscription created", zap.String("event_id", event.ID))
}

// PaymentSuccessPage handles the payment success redirect page
// @Summary Payment success page
// @Description Handle successful payment redirect from Stripe
// @Tags payments
// @Produce html
// @Param session_id query string true "Stripe session ID"
// @Success 200 {string} string "HTML success page"
// @Router /payment/success [get]
func (h *PaymentHandler) PaymentSuccessPage(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Verify session exists and get details
	session, err := session.Get(sessionID, nil)
	if err != nil {
		h.logger.Error("Failed to retrieve session", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session"})
		return
	}

	// Get booking info from metadata
	bookingID, exists := session.Metadata["booking_id"]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session metadata"})
		return
	}

	// Return simple success page (in production, this would be a proper HTML template)
	c.JSON(http.StatusOK, gin.H{
		"message":    "Payment successful!",
		"session_id": sessionID,
		"booking_id": bookingID,
		"status":     session.PaymentStatus,
	})
}

// PaymentCancelPage handles the payment cancel redirect page
// @Summary Payment cancel page
// @Description Handle cancelled payment redirect from Stripe
// @Tags payments
// @Produce html
// @Success 200 {string} string "HTML cancel page"
// @Router /payment/cancel [get]
func (h *PaymentHandler) PaymentCancelPage(c *gin.Context) {
	// Return simple cancel page (in production, this would be a proper HTML template)
	c.JSON(http.StatusOK, gin.H{
		"message": "Payment was cancelled. You can try again later.",
	})
}