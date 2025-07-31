package testutils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/database"
	"elterngeld-portal/internal/models"
	"elterngeld-portal/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TestContext holds common test dependencies
type TestContext struct {
	DB       *gorm.DB
	Config   *config.Config
	Logger   *zap.Logger
	JWTService *auth.JWTService
	TempDir  string
}

// SetupTestContext creates a complete test context with database, logger, and JWT service
func SetupTestContext(t *testing.T) *TestContext {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: dbPath,
		},
		Log: config.LogConfig{
			Level:  "silent",
			Format: "json",
		},
		Server: config.ServerConfig{
			Env: "test",
		},
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-for-jwt-tokens",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}

	// Initialize logger
	testLogger := zap.NewNop()

	// Connect to database
	err := database.Connect(cfg, testLogger)
	require.NoError(t, err)
	require.NotNil(t, database.DB)

	// Run migrations
	err = database.AutoMigrate()
	require.NoError(t, err)

	// Create JWT service
	jwtService := auth.NewJWTService(cfg)

	return &TestContext{
		DB:       database.DB,
		Config:   cfg,
		Logger:   testLogger,
		JWTService: jwtService,
		TempDir:  tempDir,
	}
}

// CleanupTestContext cleans up test resources
func CleanupTestContext(ctx *TestContext) {
	if ctx.DB != nil {
		database.Close()
		database.DB = nil
	}
	
	// Clean up SQLite file
	if ctx.Config.Database.SQLitePath != "" && ctx.Config.Database.SQLitePath != ":memory:" {
		os.Remove(ctx.Config.Database.SQLitePath)
	}
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *gorm.DB, role models.UserRole) *models.User {
	user := &models.User{
		ID:            uuid.New(),
		Email:         "test-" + uuid.New().String()[:8] + "@example.com",
		Password:      "password123",
		FirstName:     "Test",
		LastName:      "User",
		Role:          role,
		IsActive:      true,
		EmailVerified: true,
		Phone:         "+49 151 12345678",
		Address:       "Test Street 123",
		PostalCode:    "12345",
		City:          "Test City",
	}

	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// CreateTestLead creates a test lead in the database
func CreateTestLead(t *testing.T, db *gorm.DB, userID uuid.UUID, beraterID *uuid.UUID) *models.Lead {
	lead := &models.Lead{
		ID:               uuid.New(),
		UserID:           userID,
		BeraterID:        beraterID,
		Title:            "Test Lead - " + uuid.New().String()[:8],
		Description:      "This is a test lead for automated testing",
		Status:           models.LeadStatusNew,
		Priority:         models.PriorityMedium,
		ChildName:        "Test Child",
		ExpectedAmount:   1800.0,
		PreferredContact: "email",
	}

	err := db.Create(lead).Error
	require.NoError(t, err)
	return lead
}

// CreateTestDocument creates a test document in the database
func CreateTestDocument(t *testing.T, db *gorm.DB, leadID, userID uuid.UUID) *models.Document {
	document := &models.Document{
		ID:           uuid.New(),
		LeadID:       leadID,
		UserID:       userID,
		FileName:     "test-document-" + uuid.New().String()[:8] + ".pdf",
		OriginalName: "test-document.pdf",
		FilePath:     "/uploads/test-document.pdf",
		FileSize:     1024,
		ContentType:  "application/pdf",
		FileExtension: ".pdf",
		DocumentType: models.DocumentTypeApplication,
		Description:  "Test document for automated testing",
		IsProcessed:  false,
	}

	err := db.Create(document).Error
	require.NoError(t, err)
	return document
}

// CreateTestPayment creates a test payment in the database
func CreateTestPayment(t *testing.T, db *gorm.DB, leadID, userID uuid.UUID) *models.Payment {
	payment := &models.Payment{
		ID:          uuid.New(),
		LeadID:      leadID,
		UserID:      userID,
		Amount:      150.00,
		Currency:    "EUR",
		Status:      models.PaymentStatusPending,
		Method:      models.PaymentMethodStripe,
		Description: "Test payment for automated testing",
	}

	err := db.Create(payment).Error
	require.NoError(t, err)
	return payment
}

// CreateTestActivity creates a test activity in the database
func CreateTestActivity(t *testing.T, db *gorm.DB, userID, leadID uuid.UUID) *models.Activity {
	activity := &models.Activity{
		ID:          uuid.New(),
		UserID:      &userID,
		LeadID:      &leadID,
		Type:        models.ActivityTypeLeadCreated,
		Title:       "Test Activity",
		Description: "Test activity for automated testing",
		IPAddress:   "127.0.0.1",
		UserAgent:   "Test User Agent",
	}

	err := db.Create(activity).Error
	require.NoError(t, err)
	return activity
}

// CreateTestComment creates a test comment in the database
func CreateTestComment(t *testing.T, db *gorm.DB, leadID, userID uuid.UUID) *models.Comment {
	comment := &models.Comment{
		ID:         uuid.New(),
		LeadID:     leadID,
		UserID:     userID,
		Content:    "This is a test comment for automated testing",
		IsInternal: false,
	}

	err := db.Create(comment).Error
	require.NoError(t, err)
	return comment
}

// GenerateAuthToken generates a JWT token for testing
func GenerateAuthToken(t *testing.T, jwtService *auth.JWTService, user *models.User) string {
	tokenPair, err := jwtService.GenerateTokenPair(user)
	require.NoError(t, err)
	return tokenPair.AccessToken
}

// CreateAuthenticatedRequest creates an HTTP request with authentication header
func CreateAuthenticatedRequest(method, url string, body string, token string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	return req
}

// ParseJSONResponse parses JSON response body into a struct
func ParseJSONResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), target)
	require.NoError(t, err)
}

// AssertJSONResponse asserts that the response has the expected status and contains expected fields
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) {
	require.Equal(t, expectedStatus, w.Code)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	ParseJSONResponse(t, w, &response)

	for key, expectedValue := range expectedFields {
		require.Contains(t, response, key)
		if expectedValue != nil {
			require.Equal(t, expectedValue, response[key])
		}
	}
}

// AssertErrorResponse asserts that the response is an error with expected message
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	require.Equal(t, expectedStatus, w.Code)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	ParseJSONResponse(t, w, &response)

	require.Contains(t, response, "error")
	if expectedErrorMessage != "" {
		require.Contains(t, response["error"].(string), expectedErrorMessage)
	}
}

// SetupGinTestMode sets up Gin in test mode
func SetupGinTestMode() {
	gin.SetMode(gin.TestMode)
}

// CreateCompleteTestData creates a full set of related test data
func CreateCompleteTestData(t *testing.T, db *gorm.DB) *TestData {
	// Create users
	admin := CreateTestUser(t, db, models.RoleAdmin)
	berater := CreateTestUser(t, db, models.RoleBerater)
	user := CreateTestUser(t, db, models.RoleUser)

	// Create lead
	lead := CreateTestLead(t, db, user.ID, &berater.ID)

	// Create document
	document := CreateTestDocument(t, db, lead.ID, user.ID)

	// Create payment
	payment := CreateTestPayment(t, db, lead.ID, user.ID)

	// Create activity
	activity := CreateTestActivity(t, db, user.ID, lead.ID)

	// Create comment
	comment := CreateTestComment(t, db, lead.ID, user.ID)

	return &TestData{
		Admin:     admin,
		Berater:   berater,
		User:      user,
		Lead:      lead,
		Document:  document,
		Payment:   payment,
		Activity:  activity,
		Comment:   comment,
	}
}

// TestData holds a complete set of test data
type TestData struct {
	Admin     *models.User
	Berater   *models.User
	User      *models.User
	Lead      *models.Lead
	Document  *models.Document
	Payment   *models.Payment
	Activity  *models.Activity
	Comment   *models.Comment
}

// MockTime provides utilities for time-based testing
type MockTime struct {
	currentTime time.Time
}

// NewMockTime creates a new mock time instance
func NewMockTime(t time.Time) *MockTime {
	return &MockTime{currentTime: t}
}

// Now returns the current mock time
func (mt *MockTime) Now() time.Time {
	return mt.currentTime
}

// Advance advances the mock time by the given duration
func (mt *MockTime) Advance(d time.Duration) {
	mt.currentTime = mt.currentTime.Add(d)
}

// AssertRecordExists verifies that a record exists in the database
func AssertRecordExists(t *testing.T, db *gorm.DB, model interface{}, conditions ...interface{}) {
	err := db.First(model, conditions...).Error
	require.NoError(t, err, "Expected record to exist but it was not found")
}

// AssertRecordNotExists verifies that a record does not exist in the database
func AssertRecordNotExists(t *testing.T, db *gorm.DB, model interface{}, conditions ...interface{}) {
	err := db.First(model, conditions...).Error
	require.Error(t, err, "Expected record to not exist but it was found")
	require.Equal(t, gorm.ErrRecordNotFound, err)
}

// AssertRecordCount verifies the count of records matching the conditions
func AssertRecordCount(t *testing.T, db *gorm.DB, model interface{}, expectedCount int64, conditions ...interface{}) {
	var count int64
	query := db.Model(model)
	if len(conditions) > 0 {
		query = query.Where(conditions[0], conditions[1:]...)
	}
	err := query.Count(&count).Error
	require.NoError(t, err)
	require.Equal(t, expectedCount, count)
}

// CleanupDatabase removes all data from test database tables
func CleanupDatabase(t *testing.T, db *gorm.DB) {
	// Order matters due to foreign key constraints
	tables := []string{
		"activities",
		"comments",
		"documents",
		"payments",
		"leads",
		"refresh_tokens",
		"users",
	}

	for _, table := range tables {
		err := db.Exec("DELETE FROM " + table).Error
		require.NoError(t, err)
	}
}

// TestHTTPClient provides utilities for HTTP testing
type TestHTTPClient struct {
	router *gin.Engine
}

// NewTestHTTPClient creates a new test HTTP client
func NewTestHTTPClient(router *gin.Engine) *TestHTTPClient {
	return &TestHTTPClient{router: router}
}

// GET performs a GET request
func (c *TestHTTPClient) GET(url string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", url, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, req)
	return w
}

// POST performs a POST request
func (c *TestHTTPClient) POST(url string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, req)
	return w
}

// PUT performs a PUT request
func (c *TestHTTPClient) PUT(url string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("PUT", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, req)
	return w
}

// DELETE performs a DELETE request
func (c *TestHTTPClient) DELETE(url string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("DELETE", url, nil)
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, req)
	return w
}

// WithAuth adds authentication header to the request headers
func WithAuth(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(t *testing.T, uuidStr string) {
	_, err := uuid.Parse(uuidStr)
	require.NoError(t, err, "Expected valid UUID, got: %s", uuidStr)
}

// AssertTimestampRecent checks if a timestamp is within the last minute
func AssertTimestampRecent(t *testing.T, timestamp time.Time) {
	now := time.Now()
	diff := now.Sub(timestamp)
	require.True(t, diff >= 0, "Timestamp should not be in the future")
	require.True(t, diff < time.Minute, "Timestamp should be recent (within last minute)")
}

// RandomEmail generates a random email for testing
func RandomEmail() string {
	return "test-" + uuid.New().String()[:8] + "@example.com"
}

// RandomString generates a random string of specified length
func RandomString(length int) string {
	return uuid.New().String()[:length]
}