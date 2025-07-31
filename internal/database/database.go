package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes a database connection
func Connect(cfg *config.Config, zapLogger *zap.Logger) error {
	var err error
	var db *gorm.DB

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  getLogLevel(cfg.Log.Level),
			IgnoreRecordNotFoundError: true,
			Colorful:                  cfg.IsDevelopment(),
		},
	)

	// GORM configuration
	gormConfig := &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connect based on driver
	switch cfg.Database.Driver {
	case "postgres":
		dsn := cfg.GetDSN()
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		zapLogger.Info("Connected to PostgreSQL database")

	case "sqlite":
		// Ensure directory exists for SQLite
		if err := ensureDir(filepath.Dir(cfg.Database.SQLitePath)); err != nil {
			return fmt.Errorf("failed to create SQLite directory: %w", err)
		}

		db, err = gorm.Open(sqlite.Open(cfg.Database.SQLitePath), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SQLite: %w", err)
		}
		zapLogger.Info("Connected to SQLite database", zap.String("path", cfg.Database.SQLitePath))

		// Enable foreign key constraints for SQLite
		db.Exec("PRAGMA foreign_keys = ON;")

	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	// Configure connection pool for PostgreSQL
	if cfg.Database.Driver == "postgres" {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get database instance: %w", err)
		}

		// Connection pool settings
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	DB = db

	// Auto-migrate if enabled
	if cfg.Dev.AutoMigrate {
		if err := AutoMigrate(); err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}
		zapLogger.Info("Database auto-migration completed")
	}

	return nil
}

// AutoMigrate runs automatic migrations for all models
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// List of models to migrate
	models := []interface{}{
		&models.User{},
		&models.RefreshToken{},
		&models.Lead{},
		&models.Comment{},
		&models.Document{},
		&models.Activity{},
		&models.Payment{},
	}

	// Run migrations
	for _, model := range models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Create indexes manually if needed
	if err := createCustomIndexes(); err != nil {
		return fmt.Errorf("failed to create custom indexes: %w", err)
	}

	return nil
}

// createCustomIndexes creates custom database indexes
func createCustomIndexes() error {
	// Add custom indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_activities_created_at ON activities(created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_activities_type_created_at ON activities(type, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_leads_status_created_at ON leads(status, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);",
		"CREATE INDEX IF NOT EXISTS idx_documents_lead_id_created_at ON documents(lead_id, created_at DESC);",
	}

	for _, indexSQL := range indexes {
		if err := DB.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Failed to create index: %s, error: %v", indexSQL, err)
		}
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// IsHealthy checks if the database connection is healthy
func IsHealthy() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

// GetStats returns database connection statistics
func GetStats() map[string]interface{} {
	if DB == nil {
		return map[string]interface{}{
			"status": "not_initialized",
		}
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"status":           "connected",
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUse,
		"idle":            stats.Idle,
		"max_open_conns":  stats.MaxOpenConnections,
		"wait_count":      stats.WaitCount,
		"wait_duration":   stats.WaitDuration.String(),
		"max_idle_closed": stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}
}

// Transaction runs a function within a database transaction
func Transaction(fn func(*gorm.DB) error) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	return DB.Transaction(fn)
}

// Paginate creates a pagination scope for GORM queries
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		if pageSize > 100 {
			pageSize = 100
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// CalculatePagination calculates pagination info
func CalculatePagination(page, pageSize int, total int64) PaginationInfo {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

// getLogLevel converts string log level to GORM log level
func getLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Info
	}
}

// SeedData creates initial data for development
func SeedData(cfg *config.Config) error {
	if !cfg.Dev.SeedData {
		return nil
	}

	log.Println("Seeding development data...")

	// Check if admin user already exists
	var adminUser models.User
	err := DB.Where("email = ?", cfg.Admin.Email).First(&adminUser).Error
	if err == nil {
		log.Println("Admin user already exists, skipping seed data")
		return nil
	}

	// Create admin user
	admin := &models.User{
		Email:     cfg.Admin.Email,
		Password:  cfg.Admin.Password,
		FirstName: "Admin",
		LastName:  "Administrator",
		Role:      models.RoleAdmin,
		IsActive:  true,
		EmailVerified: true,
	}

	if err := DB.Create(admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Create test berater
	berater := &models.User{
		Email:     "berater@elterngeld-portal.de",
		Password:  "berater123",
		FirstName: "Max",
		LastName:  "Berater",
		Role:      models.RoleBerater,
		IsActive:  true,
		EmailVerified: true,
	}

	if err := DB.Create(berater).Error; err != nil {
		return fmt.Errorf("failed to create berater user: %w", err)
	}

	// Create test user
	user := &models.User{
		Email:     "user@example.com",
		Password:  "user123",
		FirstName: "Anna",
		LastName:  "Mustermann",
		Role:      models.RoleUser,
		IsActive:  true,
		EmailVerified: true,
		Phone:     "+49 151 12345678",
		Address:   "Musterstraße 123",
		PostalCode: "12345",
		City:      "Musterstadt",
	}

	if err := DB.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	// Create test lead
	lead := &models.Lead{
		UserID:      user.ID,
		Title:       "Elterngeld für erstes Kind",
		Description: "Antrag auf Elterngeld für unser erstes Kind. Geburt voraussichtlich im März 2024.",
		Status:      models.LeadStatusNew,
		Priority:    models.PriorityMedium,
		ChildName:   "Baby Mustermann",
		ExpectedAmount: 1800.0,
		PreferredContact: "email",
	}

	if err := DB.Create(lead).Error; err != nil {
		return fmt.Errorf("failed to create test lead: %w", err)
	}

	// Create activity for lead creation
	activity := models.CreateLeadCreatedActivity(user.ID, lead.ID, lead.Title)
	if err := DB.Create(activity).Error; err != nil {
		log.Printf("Warning: Failed to create lead creation activity: %v", err)
	}

	log.Printf("Seed data created successfully:")
	log.Printf("  Admin: %s / %s", cfg.Admin.Email, cfg.Admin.Password)
	log.Printf("  Berater: %s / berater123", berater.Email)
	log.Printf("  User: %s / user123", user.Email)

	return nil
}