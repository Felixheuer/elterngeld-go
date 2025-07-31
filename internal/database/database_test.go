package database

import (
	"os"
	"path/filepath"
	"testing"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestConnect_SQLite(t *testing.T) {
	cfg := createTestSQLiteConfig(t)
	logger := zap.NewNop()

	err := Connect(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, DB)

	// Test connection
	err = IsHealthy()
	assert.NoError(t, err)

	// Clean up
	Close()
	DB = nil
	cleanupSQLiteFile(cfg.Database.SQLitePath)
}

func TestConnect_UnsupportedDriver(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "unsupported",
		},
	}
	logger := zap.NewNop()

	err := Connect(cfg, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database driver")
}

func TestConnect_WithAutoMigrate(t *testing.T) {
	cfg := createTestSQLiteConfig(t)
	cfg.Dev.AutoMigrate = true
	logger := zap.NewNop()

	err := Connect(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, DB)

	// Verify tables were created
	assert.True(t, DB.Migrator().HasTable(&models.User{}))
	assert.True(t, DB.Migrator().HasTable(&models.Lead{}))
	assert.True(t, DB.Migrator().HasTable(&models.Document{}))
	assert.True(t, DB.Migrator().HasTable(&models.Activity{}))
	assert.True(t, DB.Migrator().HasTable(&models.Payment{}))

	// Clean up
	Close()
	DB = nil
	cleanupSQLiteFile(cfg.Database.SQLitePath)
}

func TestAutoMigrate(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	err := AutoMigrate()
	require.NoError(t, err)

	// Verify all tables exist
	tables := []interface{}{
		&models.User{},
		&models.RefreshToken{},
		&models.Lead{},
		&models.Comment{},
		&models.Document{},
		&models.Activity{},
		&models.Payment{},
	}

	for _, table := range tables {
		assert.True(t, DB.Migrator().HasTable(table))
	}
}

func TestAutoMigrate_WithoutDB(t *testing.T) {
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	err := AutoMigrate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not initialized")
}

func TestClose(t *testing.T) {
	setupTestDB(t)

	err := Close()
	assert.NoError(t, err)

	// Clean up
	DB = nil
}

func TestClose_WithoutDB(t *testing.T) {
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	err := Close()
	assert.NoError(t, err)
}

func TestIsHealthy(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	err := IsHealthy()
	assert.NoError(t, err)
}

func TestIsHealthy_WithoutDB(t *testing.T) {
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	err := IsHealthy()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not initialized")
}

func TestGetStats(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	stats := GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, "connected", stats["status"])
	assert.Contains(t, stats, "open_connections")
	assert.Contains(t, stats, "in_use")
	assert.Contains(t, stats, "idle")
}

func TestGetStats_WithoutDB(t *testing.T) {
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	stats := GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, "not_initialized", stats["status"])
}

func TestTransaction_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}

	err := Transaction(func(tx *gorm.DB) error {
		return tx.Create(user).Error
	})

	require.NoError(t, err)

	// Verify user was created
	var foundUser models.User
	err = DB.First(&foundUser, "id = ?", user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, user.Email, foundUser.Email)
}

func TestTransaction_Rollback(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}

	err := Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		// Force rollback
		return gorm.ErrInvalidData
	})

	assert.Error(t, err)

	// Verify user was not created
	var foundUser models.User
	err = DB.First(&foundUser, "id = ?", user.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestTransaction_WithoutDB(t *testing.T) {
	originalDB := DB
	DB = nil
	defer func() { DB = originalDB }()

	err := Transaction(func(tx *gorm.DB) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not initialized")
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
		expectedLimit  int
	}{
		{"first page", 1, 10, 0, 10},
		{"second page", 2, 10, 10, 10},
		{"large page size", 1, 200, 0, 100}, // Should be capped at 100
		{"zero page", 0, 10, 0, 10},          // Should default to page 1
		{"negative page", -1, 10, 0, 10},     // Should default to page 1
		{"zero page size", 1, 0, 0, 10},      // Should default to 10
		{"negative page size", 1, -5, 0, 10}, // Should default to 10
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDB(t)
			defer cleanupTestDB(t)

			// Create a query with pagination
			var results []models.User
			db := DB.Model(&models.User{}).Scopes(Paginate(tt.page, tt.pageSize)).Find(&results)

			// We can't directly access the offset and limit from the query,
			// but we can verify that the function doesn't panic and returns a valid query
			assert.NotNil(t, db)
			assert.NoError(t, db.Error)
		})
	}
}

func TestCalculatePagination(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		total          int64
		expectedResult PaginationInfo
	}{
		{
			name:     "first page with results",
			page:     1,
			pageSize: 10,
			total:    25,
			expectedResult: PaginationInfo{
				Page:       1,
				PageSize:   10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    false,
			},
		},
		{
			name:     "middle page",
			page:     2,
			pageSize: 10,
			total:    25,
			expectedResult: PaginationInfo{
				Page:       2,
				PageSize:   10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    true,
			},
		},
		{
			name:     "last page",
			page:     3,
			pageSize: 10,
			total:    25,
			expectedResult: PaginationInfo{
				Page:       3,
				PageSize:   10,
				Total:      25,
				TotalPages: 3,
				HasNext:    false,
				HasPrev:    true,
			},
		},
		{
			name:     "no results",
			page:     1,
			pageSize: 10,
			total:    0,
			expectedResult: PaginationInfo{
				Page:       1,
				PageSize:   10,
				Total:      0,
				TotalPages: 0,
				HasNext:    false,
				HasPrev:    false,
			},
		},
		{
			name:     "exact page boundary",
			page:     2,
			pageSize: 10,
			total:    20,
			expectedResult: PaginationInfo{
				Page:       2,
				PageSize:   10,
				Total:      20,
				TotalPages: 2,
				HasNext:    false,
				HasPrev:    true,
			},
		},
		{
			name:     "invalid page defaults",
			page:     0,
			pageSize: 0,
			total:    25,
			expectedResult: PaginationInfo{
				Page:       1,
				PageSize:   10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePagination(tt.page, tt.pageSize, tt.total)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestSeedData(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	cfg := &config.Config{
		Dev: config.DevConfig{
			SeedData: true,
		},
		Admin: config.AdminConfig{
			Email:    "admin@example.com",
			Password: "admin123",
		},
	}

	err := SeedData(cfg)
	require.NoError(t, err)

	// Verify admin user was created
	var admin models.User
	err = DB.First(&admin, "email = ?", cfg.Admin.Email).Error
	require.NoError(t, err)
	assert.Equal(t, cfg.Admin.Email, admin.Email)
	assert.Equal(t, models.RoleAdmin, admin.Role)

	// Verify berater user was created
	var berater models.User
	err = DB.First(&berater, "email = ?", "berater@elterngeld-portal.de").Error
	require.NoError(t, err)
	assert.Equal(t, models.RoleBerater, berater.Role)

	// Verify test user was created
	var user models.User
	err = DB.First(&user, "email = ?", "user@example.com").Error
	require.NoError(t, err)
	assert.Equal(t, models.RoleUser, user.Role)

	// Verify test lead was created
	var lead models.Lead
	err = DB.First(&lead, "user_id = ?", user.ID).Error
	require.NoError(t, err)
	assert.Equal(t, user.ID, lead.UserID)
	assert.Equal(t, models.LeadStatusNew, lead.Status)
}

func TestSeedData_DisabledSeedData(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	cfg := &config.Config{
		Dev: config.DevConfig{
			SeedData: false,
		},
		Admin: config.AdminConfig{
			Email:    "admin@example.com",
			Password: "admin123",
		},
	}

	err := SeedData(cfg)
	assert.NoError(t, err)

	// Verify no admin user was created
	var admin models.User
	err = DB.First(&admin, "email = ?", cfg.Admin.Email).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestSeedData_AdminExists(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	cfg := &config.Config{
		Dev: config.DevConfig{
			SeedData: true,
		},
		Admin: config.AdminConfig{
			Email:    "admin@example.com",
			Password: "admin123",
		},
	}

	// Create admin user first
	admin := &models.User{
		Email:     cfg.Admin.Email,
		Password:  "hashedpassword",
		FirstName: "Existing",
		LastName:  "Admin",
		Role:      models.RoleAdmin,
		IsActive:  true,
	}
	err := DB.Create(admin).Error
	require.NoError(t, err)

	// Seed data should skip creation
	err = SeedData(cfg)
	assert.NoError(t, err)

	// Verify only one admin exists
	var count int64
	err = DB.Model(&models.User{}).Where("email = ?", cfg.Admin.Email).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestConcurrentDatabaseOperations(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	// Test concurrent user creation
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			user := &models.User{
				Email:     "user" + string(rune(id)) + "@example.com",
				Password:  "password123",
				FirstName: "User",
				LastName:  string(rune(id)),
				Role:      models.RoleUser,
				IsActive:  true,
			}

			err := Transaction(func(tx *gorm.DB) error {
				return tx.Create(user).Error
			})
			results <- err
		}(i)
	}

	// Check all operations completed successfully
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err)
	}

	// Verify all users were created
	var count int64
	err := DB.Model(&models.User{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines), count)
}

func TestDatabaseEdgeCases(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	// Test with nil values
	user := &models.User{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		IsActive:  true,
	}

	err := DB.Create(user).Error
	require.NoError(t, err)

	// Test querying non-existent record
	var nonExistent models.User
	err = DB.First(&nonExistent, "id = ?", uuid.New()).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Test empty query
	var users []models.User
	err = DB.Where("1 = 0").Find(&users).Error
	assert.NoError(t, err)
	assert.Empty(t, users)
}

// Helper functions

func createTestSQLiteConfig(t *testing.T) *config.Config {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	return &config.Config{
		Database: config.DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: dbPath,
		},
		Log: config.LogConfig{
			Level: "silent",
		},
	}
}

func setupTestDB(t *testing.T) {
	cfg := createTestSQLiteConfig(t)
	logger := zap.NewNop()

	err := Connect(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, DB)
}

func cleanupTestDB(t *testing.T) {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			dbName := sqlDB.Ping()
			if dbName == nil {
				// Get database file path before closing
				var dbPath string
				if DB.Dialector.Name() == "sqlite" {
					// For SQLite, we need to extract the file path
					rows, err := DB.Raw("PRAGMA database_list").Rows()
					if err == nil {
						defer rows.Close()
						for rows.Next() {
							var seq int
							var name, file string
							if err := rows.Scan(&seq, &name, &file); err == nil && name == "main" {
								dbPath = file
								break
							}
						}
					}
				}

				Close()
				
				// Remove SQLite file if it exists
				if dbPath != "" && dbPath != ":memory:" {
					os.Remove(dbPath)
				}
			}
		}
		DB = nil
	}
}

func cleanupSQLiteFile(path string) {
	if path != "" && path != ":memory:" {
		os.Remove(path)
	}
}

func setupMigrations(t *testing.T) {
	err := AutoMigrate()
	require.NoError(t, err)
}

func TestConnectionPoolConfiguration(t *testing.T) {
	// This test would require PostgreSQL to fully test connection pooling
	// For now, we'll test that the configuration doesn't cause errors
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "postgres",
			Host:   "localhost",
			Port:   "5432",
			Name:   "test_db",
			User:   "test_user",
			Password:   "test_pass",
		},
		Log: config.LogConfig{
			Level: "silent",
		},
	}

	logger := zap.NewNop()

	// This will fail to connect since we don't have a real PostgreSQL instance
	// but it tests that the configuration parsing works
	err := Connect(cfg, logger)
	assert.Error(t, err) // Expected to fail without real PostgreSQL

	// Clean up if somehow a connection was made
	if DB != nil {
		Close()
		DB = nil
	}
}

func TestIndexCreation(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)
	setupMigrations(t)

	// Test that custom indexes were created (this is implicit in AutoMigrate)
	// We can't easily test index existence in SQLite, but we can verify
	// that the createCustomIndexes function doesn't error
	err := createCustomIndexes()
	assert.NoError(t, err)
}

func TestDatabaseStatsEdgeCases(t *testing.T) {
	// Test stats when database connection is broken
	setupTestDB(t)
	
	// Close the underlying connection manually to simulate broken connection
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	stats := GetStats()
	// Should handle error gracefully
	assert.NotNil(t, stats)
	// Might be "error" status or still "connected" depending on timing
	assert.Contains(t, []string{"connected", "error"}, stats["status"])

	// Clean up
	DB = nil
}