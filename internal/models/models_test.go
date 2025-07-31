package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestUserModel_HashPassword(t *testing.T) {
	user := &User{}

	t.Run("hashes_valid_password", func(t *testing.T) {
		user.Password = "testpassword123"
		err := user.HashPassword()
		
		assert.NoError(t, err)
		assert.NotEqual(t, "testpassword123", user.Password)
		assert.Greater(t, len(user.Password), 50)

		// Should be valid bcrypt hash
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("testpassword123"))
		assert.NoError(t, err)
	})

	t.Run("handles_empty_password", func(t *testing.T) {
		user.Password = ""
		err := user.HashPassword()
		
		assert.NoError(t, err)
		assert.Equal(t, "", user.Password)
	})
}

func TestUserModel_CheckPassword(t *testing.T) {
	user := &User{
		Password: "plainpassword",
	}
	user.HashPassword()

	t.Run("correct_password", func(t *testing.T) {
		assert.True(t, user.CheckPassword("plainpassword"))
	})

	t.Run("incorrect_password", func(t *testing.T) {
		assert.False(t, user.CheckPassword("wrongpassword"))
	})

	t.Run("empty_password", func(t *testing.T) {
		assert.False(t, user.CheckPassword(""))
	})

	t.Run("with_unhashed_password", func(t *testing.T) {
		userPlain := &User{Password: "plaintext"}
		assert.False(t, userPlain.CheckPassword("plaintext"))
	})
}

func TestUserModel_ToResponse(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		FirstName:     "Test",
		LastName:      "User",
		Phone:         "+49 151 12345678",
		Role:          RoleUser,
		IsActive:      true,
		DateOfBirth:   &now,
		Address:       "Test Street 123",
		PostalCode:    "12345",
		City:          "Test City",
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
		Password:      "should-not-be-included",
		ResetToken:    "should-not-be-included",
	}

	response := user.ToResponse()

	// Should include public fields
	assert.Equal(t, user.ID, response.ID)
	assert.Equal(t, user.Email, response.Email)
	assert.Equal(t, user.FirstName, response.FirstName)
	assert.Equal(t, user.LastName, response.LastName)
	assert.Equal(t, user.Phone, response.Phone)
	assert.Equal(t, user.Role, response.Role)
	assert.Equal(t, user.IsActive, response.IsActive)
	assert.Equal(t, user.DateOfBirth, response.DateOfBirth)
	assert.Equal(t, user.Address, response.Address)
	assert.Equal(t, user.PostalCode, response.PostalCode)
	assert.Equal(t, user.City, response.City)
	assert.Equal(t, user.EmailVerified, response.EmailVerified)
	assert.Equal(t, user.CreatedAt, response.CreatedAt)
	assert.Equal(t, user.UpdatedAt, response.UpdatedAt)
}

func TestUserModel_FullName(t *testing.T) {
	user := &User{
		FirstName: "John",
		LastName:  "Doe",
	}

	assert.Equal(t, "John Doe", user.FullName())
}

func TestUserModel_RoleChecks(t *testing.T) {
	tests := []struct {
		name      string
		role      UserRole
		isAdmin   bool
		isBerater bool
		isUser    bool
	}{
		{"admin_user", RoleAdmin, true, false, false},
		{"berater_user", RoleBerater, false, true, false},
		{"regular_user", RoleUser, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			assert.Equal(t, tt.isAdmin, user.IsAdmin())
			assert.Equal(t, tt.isBerater, user.IsBerater())
			assert.Equal(t, tt.isUser, user.IsUser())
		})
	}
}

func TestUserRoles(t *testing.T) {
	assert.Equal(t, UserRole("user"), RoleUser)
	assert.Equal(t, UserRole("berater"), RoleBerater)
	assert.Equal(t, UserRole("admin"), RoleAdmin)
}

func TestLeadModel_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name         string
		currentStatus LeadStatus
		targetStatus  LeadStatus
		canTransition bool
	}{
		{"new_to_in_progress", LeadStatusNew, LeadStatusInProgress, true},
		{"new_to_payment_pending", LeadStatusNew, LeadStatusPaymentPending, true},
		{"new_to_cancelled", LeadStatusNew, LeadStatusCancelled, true},
		{"new_to_completed", LeadStatusNew, LeadStatusCompleted, false},
		{"in_progress_to_question", LeadStatusInProgress, LeadStatusQuestion, true},
		{"in_progress_to_completed", LeadStatusInProgress, LeadStatusCompleted, true},
		{"completed_to_new", LeadStatusCompleted, LeadStatusNew, false},
		{"cancelled_to_new", LeadStatusCancelled, LeadStatusNew, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lead := &Lead{Status: tt.currentStatus}
			assert.Equal(t, tt.canTransition, lead.CanTransitionTo(tt.targetStatus))
		})
	}
}

func TestLeadModel_StatusChecks(t *testing.T) {
	tests := []struct {
		name          string
		status        LeadStatus
		isCompleted   bool
		isCancelled   bool
		isActive      bool
		needsPayment  bool
	}{
		{"new", LeadStatusNew, false, false, true, false},
		{"in_progress", LeadStatusInProgress, false, false, true, false},
		{"completed", LeadStatusCompleted, true, false, false, false},
		{"cancelled", LeadStatusCancelled, false, true, false, false},
		{"payment_pending", LeadStatusPaymentPending, false, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lead := &Lead{Status: tt.status}
			assert.Equal(t, tt.isCompleted, lead.IsCompleted())
			assert.Equal(t, tt.isCancelled, lead.IsCancelled())
			assert.Equal(t, tt.isActive, lead.IsActive())
			assert.Equal(t, tt.needsPayment, lead.NeedsPayment())
		})
	}
}

func TestLeadModel_IsOverdue(t *testing.T) {
	pastDate := time.Now().Add(-24 * time.Hour)
	futureDate := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name      string
		lead      *Lead
		isOverdue bool
	}{
		{"no_due_date", &Lead{Status: LeadStatusInProgress}, false},
		{"past_due_date_active", &Lead{Status: LeadStatusInProgress, DueDate: &pastDate}, true},
		{"future_due_date_active", &Lead{Status: LeadStatusInProgress, DueDate: &futureDate}, false},
		{"past_due_date_completed", &Lead{Status: LeadStatusCompleted, DueDate: &pastDate}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isOverdue, tt.lead.IsOverdue())
		})
	}
}

func TestPaymentModel_StatusChecks(t *testing.T) {
	tests := []struct {
		name         string
		status       PaymentStatus
		refundAmount float64
		isPaid       bool
		isFailed     bool
		isRefunded   bool
		isPending    bool
		canRefund    bool
	}{
		{"pending", PaymentStatusPending, 0, false, false, false, true, false},
		{"processing", PaymentStatusProcessing, 0, false, false, false, true, false},
		{"succeeded", PaymentStatusSucceeded, 0, true, false, false, false, true},
		{"failed", PaymentStatusFailed, 0, false, true, false, false, false},
		{"refunded", PaymentStatusRefunded, 100, false, false, true, false, false},
		{"succeeded_partial_refund", PaymentStatusSucceeded, 50, true, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := &Payment{
				Status:       tt.status,
				Amount:       100.0,
				RefundAmount: tt.refundAmount,
			}
			assert.Equal(t, tt.isPaid, payment.IsPaid())
			assert.Equal(t, tt.isFailed, payment.IsFailed())
			assert.Equal(t, tt.isRefunded, payment.IsRefunded())
			assert.Equal(t, tt.isPending, payment.IsPending())
			assert.Equal(t, tt.canRefund, payment.CanBeRefunded())
		})
	}
}

func TestPaymentModel_FormatAmount(t *testing.T) {
	payment := &Payment{
		Amount:   150.50,
		Currency: "EUR",
	}

	assert.Equal(t, "150.50 EUR", payment.FormatAmount())
}

func TestPaymentModel_GetRemainingRefundAmount(t *testing.T) {
	payment := &Payment{
		Status:       PaymentStatusSucceeded,
		Amount:       100.0,
		RefundAmount: 30.0,
	}

	assert.Equal(t, 70.0, payment.GetRemainingRefundAmount())
}

func TestDocumentModel_FileTypeChecks(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		isImage     bool
		isPDF       bool
		isValid     bool
	}{
		{"jpeg", "image/jpeg", true, false, true},
		{"png", "image/png", true, false, true},
		{"pdf", "application/pdf", false, true, true},
		{"text", "text/plain", false, false, false},
		{"gif", "image/gif", true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{ContentType: tt.contentType}
			assert.Equal(t, tt.isImage, doc.IsImage())
			assert.Equal(t, tt.isPDF, doc.IsPDF())
			assert.Equal(t, tt.isValid, doc.IsValid())
		})
	}
}

func TestDocumentModel_GetHumanReadableSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"bytes", 500, "500 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 2097152, "2.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{FileSize: tt.size}
			assert.Equal(t, tt.expected, doc.GetHumanReadableSize())
		})
	}
}

func TestDocumentType_DisplayName(t *testing.T) {
	tests := []struct {
		docType  DocumentType
		expected string
	}{
		{DocumentTypeBirthCertificate, "Geburtsurkunde"},
		{DocumentTypeIncomeProof, "Einkommensnachweis"},
		{DocumentTypeEmploymentCert, "Arbeitsbescheinigung"},
		{DocumentTypeApplication, "Antrag"},
		{DocumentTypeOther, "Sonstiges"},
		{DocumentType("unknown"), "Unbekannt"},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.docType.DisplayName())
		})
	}
}

func TestActivityType_GetDisplayName(t *testing.T) {
	tests := []struct {
		activityType ActivityType
		expected     string
	}{
		{ActivityTypeLeadCreated, "Lead erstellt"},
		{ActivityTypeUserLogin, "Benutzer angemeldet"},
		{ActivityTypePaymentCompleted, "Zahlung abgeschlossen"},
		{ActivityType("unknown"), "Unbekannte Aktivität"},
	}

	for _, tt := range tests {
		t.Run(string(tt.activityType), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.activityType.GetDisplayName())
		})
	}
}

func TestActivityModel_SetAndGetMetadata(t *testing.T) {
	activity := &Activity{}
	
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	err := activity.SetMetadata(metadata)
	assert.NoError(t, err)
	assert.NotNil(t, activity.Metadata)

	var retrieved map[string]interface{}
	err = activity.GetMetadata(&retrieved)
	assert.NoError(t, err)
	assert.Equal(t, "value1", retrieved["key1"])
	assert.Equal(t, float64(42), retrieved["key2"]) // JSON unmarshaling returns float64 for numbers
}

func TestLeadModel_GenerateApplicationNumber(t *testing.T) {
	lead := &Lead{
		ID: uuid.New(),
	}

	appNumber := lead.generateApplicationNumber()
	assert.Contains(t, appNumber, "EG-")
	currentYear := time.Now().Format("2006")
	assert.Contains(t, appNumber, currentYear) // Current year
	assert.Greater(t, len(appNumber), 10)
}

func TestPaymentStatus_GetDisplayName(t *testing.T) {
	tests := []struct {
		status   PaymentStatus
		expected string
	}{
		{PaymentStatusPending, "Ausstehend"},
		{PaymentStatusSucceeded, "Erfolgreich"},
		{PaymentStatusFailed, "Fehlgeschlagen"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.GetDisplayName())
		})
	}
}

func TestPaymentMethod_GetDisplayName(t *testing.T) {
	tests := []struct {
		method   PaymentMethod
		expected string
	}{
		{PaymentMethodStripe, "Kreditkarte/Online"},
		{PaymentMethodBank, "Banküberweisung"},
		{PaymentMethodCash, "Bar"},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.method.GetDisplayName())
		})
	}
}