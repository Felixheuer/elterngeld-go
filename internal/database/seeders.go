package database

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"elterngeld-portal/internal/models"
)

// SeedDatabase seeds the database with initial data
func SeedDatabase(db *gorm.DB) error {
	log.Println("Starting database seeding...")

	// Seed in order of dependencies
	if err := seedUsers(db); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	if err := seedPermissions(db); err != nil {
		return fmt.Errorf("failed to seed permissions: %w", err)
	}

	if err := seedRoles(db); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	if err := seedPackages(db); err != nil {
		return fmt.Errorf("failed to seed packages: %w", err)
	}

	if err := seedAddons(db); err != nil {
		return fmt.Errorf("failed to seed addons: %w", err)
	}

	if err := seedPackageAddons(db); err != nil {
		return fmt.Errorf("failed to seed package addons: %w", err)
	}

	if err := seedTimeslots(db); err != nil {
		return fmt.Errorf("failed to seed timeslots: %w", err)
	}

	if err := seedLeads(db); err != nil {
		return fmt.Errorf("failed to seed leads: %w", err)
	}

	if err := seedBookings(db); err != nil {
		return fmt.Errorf("failed to seed bookings: %w", err)
	}

	if err := seedTodos(db); err != nil {
		return fmt.Errorf("failed to seed todos: %w", err)
	}

	if err := seedContactForms(db); err != nil {
		return fmt.Errorf("failed to seed contact forms: %w", err)
	}

	if err := seedJobs(db); err != nil {
		return fmt.Errorf("failed to seed jobs: %w", err)
	}

	if err := seedJobApplications(db); err != nil {
		return fmt.Errorf("failed to seed job applications: %w", err)
	}

	if err := seedNotificationPreferences(db); err != nil {
		return fmt.Errorf("failed to seed notification preferences: %w", err)
	}

	log.Println("Database seeding completed successfully!")
	return nil
}

func seedUsers(db *gorm.DB) error {
	log.Println("Seeding users...")

	users := []models.User{
		{
			ID:        uuid.New(),
			Email:     "admin@elterngeld-portal.de",
			Password:  "admin123",
			FirstName: "Max",
			LastName:  "Administrator",
			Phone:     "+49 30 12345678",
			Role:      models.RoleAdmin,
			IsActive:  true,
			Address:   "Musterstraße 1",
			PostalCode: "10115",
			City:      "Berlin",
			EmailVerified: true,
			EmailVerifiedAt: func() *time.Time { t := time.Now(); return &t }(),
		},
		{
			ID:        uuid.New(),
			Email:     "berater@elterngeld-portal.de",
			Password:  "berater123",
			FirstName: "Anna",
			LastName:  "Müller",
			Phone:     "+49 30 87654321",
			Role:      models.RoleBerater,
			IsActive:  true,
			Address:   "Beratergasse 5",
			PostalCode: "10117",
			City:      "Berlin",
			EmailVerified: true,
			EmailVerifiedAt: func() *time.Time { t := time.Now(); return &t }(),
		},
		{
			ID:        uuid.New(),
			Email:     "junior@elterngeld-portal.de",
			Password:  "junior123",
			FirstName: "Tom",
			LastName:  "Schmidt",
			Phone:     "+49 30 11111111",
			Role:      models.RoleJuniorBerater,
			IsActive:  true,
			Address:   "Juniorstraße 10",
			PostalCode: "10119",
			City:      "Berlin",
			EmailVerified: true,
			EmailVerifiedAt: func() *time.Time { t := time.Now(); return &t }(),
		},
		{
			ID:        uuid.New(),
			Email:     "user@example.com",
			Password:  "user123",
			FirstName: "Lisa",
			LastName:  "Schneider",
			Phone:     "+49 30 22222222",
			Role:      models.RoleUser,
			IsActive:  true,
			Address:   "Kundenweg 15",
			PostalCode: "10179",
			City:      "Berlin",
			DateOfBirth: func() *time.Time { t := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC); return &t }(),
			EmailVerified: true,
			EmailVerifiedAt: func() *time.Time { t := time.Now(); return &t }(),
		},
		{
			ID:        uuid.New(),
			Email:     "maria.weber@example.com",
			Password:  "maria123",
			FirstName: "Maria",
			LastName:  "Weber",
			Phone:     "+49 30 33333333",
			Role:      models.RoleUser,
			IsActive:  true,
			Address:   "Familienallee 20",
			PostalCode: "10245",
			City:      "Berlin",
			DateOfBirth: func() *time.Time { t := time.Date(1985, 8, 22, 0, 0, 0, 0, time.UTC); return &t }(),
			EmailVerified: false,
		},
		{
			ID:        uuid.New(),
			Email:     "stefan.braun@example.com",
			Password:  "stefan123",
			FirstName: "Stefan",
			LastName:  "Braun",
			Phone:     "+49 30 44444444",
			Role:      models.RoleUser,
			IsActive:  true,
			Address:   "Vaterstraße 8",
			PostalCode: "10315",
			City:      "Berlin",
			DateOfBirth: func() *time.Time { t := time.Date(1988, 12, 3, 0, 0, 0, 0, time.UTC); return &t }(),
			EmailVerified: true,
			EmailVerifiedAt: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
		},
	}

	// Hash passwords
	for i := range users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(users[i].Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		users[i].Password = string(hashedPassword)
		users[i].CreatedAt = time.Now()
		users[i].UpdatedAt = time.Now()
	}

	return db.Create(&users).Error
}

func seedPermissions(db *gorm.DB) error {
	log.Println("Seeding permissions...")

	permissions := []models.Permission{
		// Dashboard permissions
		{Name: "dashboard.read", Resource: models.PermissionResourceDashboard, Action: models.PermissionActionRead, Description: "Zugriff auf das Dashboard"},
		{Name: "dashboard.admin.read", Resource: models.PermissionResourceAdminDashboard, Action: models.PermissionActionRead, Description: "Zugriff auf das Admin Dashboard"},
		{Name: "dashboard.user.read", Resource: models.PermissionResourceUserDashboard, Action: models.PermissionActionRead, Description: "Zugriff auf das User Dashboard"},

		// User management
		{Name: "users.read", Resource: models.PermissionResourceUsers, Action: models.PermissionActionRead, Description: "Benutzer anzeigen"},
		{Name: "users.create", Resource: models.PermissionResourceUsers, Action: models.PermissionActionCreate, Description: "Benutzer erstellen"},
		{Name: "users.update", Resource: models.PermissionResourceUsers, Action: models.PermissionActionUpdate, Description: "Benutzer bearbeiten"},
		{Name: "users.delete", Resource: models.PermissionResourceUsers, Action: models.PermissionActionDelete, Description: "Benutzer löschen"},
		{Name: "users.manage", Resource: models.PermissionResourceUsers, Action: models.PermissionActionManage, Description: "Vollzugriff auf Benutzerverwaltung"},
		{Name: "profile.read", Resource: models.PermissionResourceProfile, Action: models.PermissionActionRead, Description: "Eigenes Profil anzeigen"},
		{Name: "profile.update", Resource: models.PermissionResourceProfile, Action: models.PermissionActionUpdate, Description: "Eigenes Profil bearbeiten"},

		// Lead management
		{Name: "leads.read", Resource: models.PermissionResourceLeads, Action: models.PermissionActionRead, Description: "Leads anzeigen"},
		{Name: "leads.create", Resource: models.PermissionResourceLeads, Action: models.PermissionActionCreate, Description: "Leads erstellen"},
		{Name: "leads.update", Resource: models.PermissionResourceLeads, Action: models.PermissionActionUpdate, Description: "Leads bearbeiten"},
		{Name: "leads.delete", Resource: models.PermissionResourceLeads, Action: models.PermissionActionDelete, Description: "Leads löschen"},
		{Name: "leads.own.read", Resource: models.PermissionResourceOwnLeads, Action: models.PermissionActionRead, Description: "Eigene Leads anzeigen"},
		{Name: "leads.own.update", Resource: models.PermissionResourceOwnLeads, Action: models.PermissionActionUpdate, Description: "Eigene Leads bearbeiten"},
		{Name: "leads.own.manage", Resource: models.PermissionResourceOwnLeads, Action: models.PermissionActionManage, Description: "Vollzugriff auf eigene Leads"},
		{Name: "leads.all.read", Resource: models.PermissionResourceAllLeads, Action: models.PermissionActionRead, Description: "Alle Leads anzeigen"},
		{Name: "leads.all.manage", Resource: models.PermissionResourceAllLeads, Action: models.PermissionActionManage, Description: "Vollzugriff auf alle Leads"},

		// Booking management
		{Name: "bookings.read", Resource: models.PermissionResourceBookings, Action: models.PermissionActionRead, Description: "Buchungen anzeigen"},
		{Name: "bookings.create", Resource: models.PermissionResourceBookings, Action: models.PermissionActionCreate, Description: "Buchungen erstellen"},
		{Name: "bookings.update", Resource: models.PermissionResourceBookings, Action: models.PermissionActionUpdate, Description: "Buchungen bearbeiten"},
		{Name: "bookings.delete", Resource: models.PermissionResourceBookings, Action: models.PermissionActionDelete, Description: "Buchungen löschen"},
		{Name: "bookings.own.read", Resource: models.PermissionResourceOwnBookings, Action: models.PermissionActionRead, Description: "Eigene Buchungen anzeigen"},
		{Name: "bookings.own.create", Resource: models.PermissionResourceOwnBookings, Action: models.PermissionActionCreate, Description: "Eigene Buchungen erstellen"},
		{Name: "bookings.own.update", Resource: models.PermissionResourceOwnBookings, Action: models.PermissionActionUpdate, Description: "Eigene Buchungen bearbeiten"},

		// Package management
		{Name: "packages.read", Resource: models.PermissionResourcePackages, Action: models.PermissionActionRead, Description: "Pakete anzeigen"},
		{Name: "packages.create", Resource: models.PermissionResourcePackages, Action: models.PermissionActionCreate, Description: "Pakete erstellen"},
		{Name: "packages.update", Resource: models.PermissionResourcePackages, Action: models.PermissionActionUpdate, Description: "Pakete bearbeiten"},
		{Name: "packages.delete", Resource: models.PermissionResourcePackages, Action: models.PermissionActionDelete, Description: "Pakete löschen"},
		{Name: "packages.manage", Resource: models.PermissionResourcePackages, Action: models.PermissionActionManage, Description: "Vollzugriff auf Pakete"},

		// Calendar management
		{Name: "calendar.own.read", Resource: models.PermissionResourceOwnCalendar, Action: models.PermissionActionRead, Description: "Eigenen Kalender anzeigen"},
		{Name: "calendar.own.manage", Resource: models.PermissionResourceOwnCalendar, Action: models.PermissionActionManage, Description: "Vollzugriff auf eigenen Kalender"},
		{Name: "timeslots.own.read", Resource: models.PermissionResourceTimeslots, Action: models.PermissionActionRead, Description: "Eigene Timeslots anzeigen"},
		{Name: "timeslots.own.manage", Resource: models.PermissionResourceTimeslots, Action: models.PermissionActionManage, Description: "Vollzugriff auf eigene Timeslots"},

		// Todo management
		{Name: "todos.read", Resource: models.PermissionResourceTodos, Action: models.PermissionActionRead, Description: "Todos anzeigen"},
		{Name: "todos.create", Resource: models.PermissionResourceTodos, Action: models.PermissionActionCreate, Description: "Todos erstellen"},
		{Name: "todos.update", Resource: models.PermissionResourceTodos, Action: models.PermissionActionUpdate, Description: "Todos bearbeiten"},
		{Name: "todos.own.read", Resource: models.PermissionResourceOwnTodos, Action: models.PermissionActionRead, Description: "Eigene Todos anzeigen"},
		{Name: "todos.own.update", Resource: models.PermissionResourceOwnTodos, Action: models.PermissionActionUpdate, Description: "Eigene Todos bearbeiten"},

		// Document management
		{Name: "documents.read", Resource: models.PermissionResourceDocuments, Action: models.PermissionActionRead, Description: "Dokumente anzeigen"},
		{Name: "documents.create", Resource: models.PermissionResourceDocuments, Action: models.PermissionActionCreate, Description: "Dokumente erstellen"},
		{Name: "documents.own.read", Resource: models.PermissionResourceDocuments, Action: models.PermissionActionRead, Description: "Eigene Dokumente anzeigen"},
		{Name: "documents.own.create", Resource: models.PermissionResourceDocuments, Action: models.PermissionActionCreate, Description: "Eigene Dokumente erstellen"},

		// Contact forms
		{Name: "contact_forms.read", Resource: models.PermissionResourceContactForms, Action: models.PermissionActionRead, Description: "Kontaktformulare anzeigen"},
		{Name: "contact_forms.update", Resource: models.PermissionResourceContactForms, Action: models.PermissionActionUpdate, Description: "Kontaktformulare bearbeiten"},
		{Name: "contact_form.create", Resource: models.PermissionResourceContactForm, Action: models.PermissionActionCreate, Description: "Kontaktformular senden"},

		// Job management
		{Name: "jobs.read", Resource: models.PermissionResourceJobs, Action: models.PermissionActionRead, Description: "Stellenangebote anzeigen"},
		{Name: "jobs.create", Resource: models.PermissionResourceJobs, Action: models.PermissionActionCreate, Description: "Stellenangebote erstellen"},
		{Name: "jobs.update", Resource: models.PermissionResourceJobs, Action: models.PermissionActionUpdate, Description: "Stellenangebote bearbeiten"},
		{Name: "jobs.delete", Resource: models.PermissionResourceJobs, Action: models.PermissionActionDelete, Description: "Stellenangebote löschen"},
		{Name: "jobs.manage", Resource: models.PermissionResourceJobs, Action: models.PermissionActionManage, Description: "Vollzugriff auf Stellenangebote"},

		// Settings
		{Name: "settings.read", Resource: models.PermissionResourceSettings, Action: models.PermissionActionRead, Description: "Einstellungen anzeigen"},
		{Name: "settings.update", Resource: models.PermissionResourceSettings, Action: models.PermissionActionUpdate, Description: "Einstellungen bearbeiten"},
		{Name: "settings.system.manage", Resource: models.PermissionResourceSystemSettings, Action: models.PermissionActionManage, Description: "Systemeinstellungen verwalten"},
		{Name: "settings.security.manage", Resource: models.PermissionResourceSecuritySettings, Action: models.PermissionActionManage, Description: "Sicherheitseinstellungen verwalten"},

		// Payments
		{Name: "payments.read", Resource: models.PermissionResourcePayments, Action: models.PermissionActionRead, Description: "Zahlungen anzeigen"},
		{Name: "payments.manage", Resource: models.PermissionResourcePayments, Action: models.PermissionActionManage, Description: "Zahlungen verwalten"},
	}

	for i := range permissions {
		permissions[i].ID = uuid.New()
		permissions[i].IsActive = true
		permissions[i].CreatedAt = time.Now()
		permissions[i].UpdatedAt = time.Now()
	}

	return db.Create(&permissions).Error
}

func seedRoles(db *gorm.DB) error {
	log.Println("Seeding roles...")

	// Get admin user for assignments
	var adminUser models.User
	if err := db.Where("email = ?", "admin@elterngeld-portal.de").First(&adminUser).Error; err != nil {
		return err
	}

	roles := []models.Role{
		{
			ID:          uuid.New(),
			Name:        "user",
			DisplayName: "Benutzer",
			Description: "Standard-Benutzer mit eingeschränkten Rechten",
			IsActive:    true,
			IsDefault:   true,
			SortOrder:   1,
		},
		{
			ID:          uuid.New(),
			Name:        "junior_berater",
			DisplayName: "Junior Berater",
			Description: "Junior Berater mit eingeschränkten Berater-Rechten",
			IsActive:    true,
			IsDefault:   false,
			SortOrder:   2,
		},
		{
			ID:          uuid.New(),
			Name:        "berater",
			DisplayName: "Berater",
			Description: "Vollwertiger Berater mit umfangreichen Rechten",
			IsActive:    true,
			IsDefault:   false,
			SortOrder:   3,
		},
		{
			ID:          uuid.New(),
			Name:        "admin",
			DisplayName: "Administrator",
			Description: "Administrator mit vollen Rechten",
			IsActive:    true,
			IsDefault:   false,
			SortOrder:   4,
		},
	}

	for i := range roles {
		roles[i].CreatedAt = time.Now()
		roles[i].UpdatedAt = time.Now()
	}

	if err := db.Create(&roles).Error; err != nil {
		return err
	}

	// Assign permissions to roles
	return assignPermissionsToRoles(db, adminUser.ID)
}

func assignPermissionsToRoles(db *gorm.DB, adminID uuid.UUID) error {
	// Get all permissions
	var permissions []models.Permission
	if err := db.Find(&permissions).Error; err != nil {
		return err
	}

	// Get all roles
	var roles []models.Role
	if err := db.Find(&roles).Error; err != nil {
		return err
	}

	// Create permission maps for easy lookup
	permissionMap := make(map[string]models.Permission)
	for _, p := range permissions {
		permissionMap[p.Name] = p
	}

	roleMap := make(map[string]models.Role)
	for _, r := range roles {
		roleMap[r.Name] = r
	}

	// Define role permissions
	rolePermissions := map[string][]string{
		"user": {
			"dashboard.user.read", "profile.read", "profile.update",
			"bookings.own.read", "bookings.own.create", "bookings.own.update",
			"todos.own.read", "todos.own.update",
			"documents.own.read", "documents.own.create",
			"packages.read", "contact_form.create",
		},
		"junior_berater": {
			"dashboard.read", "profile.read", "profile.update",
			"leads.read", "leads.own.update",
			"bookings.read", "calendar.own.read", "timeslots.own.read",
			"todos.read", "documents.read", "contact_forms.read",
			"users.read", "packages.read",
		},
		"berater": {
			"dashboard.read", "profile.read", "profile.update",
			"leads.read", "leads.create", "leads.update", "leads.own.manage",
			"bookings.read", "bookings.create", "bookings.update",
			"calendar.own.manage", "timeslots.own.manage",
			"todos.create", "todos.update", "todos.read",
			"documents.read", "documents.create",
			"contact_forms.read", "contact_forms.update",
			"users.read", "packages.read",
		},
		"admin": {
			"dashboard.admin.read", "dashboard.read",
			"users.manage", "profile.read", "profile.update",
			"leads.all.manage", "bookings.manage",
			"packages.manage", "jobs.manage",
			"settings.system.manage", "settings.security.manage",
			"payments.manage", "contact_forms.read", "contact_forms.update",
		},
	}

	// Create role-permission assignments
	for roleName, permissionNames := range rolePermissions {
		role, exists := roleMap[roleName]
		if !exists {
			continue
		}

		for _, permissionName := range permissionNames {
			permission, exists := permissionMap[permissionName]
			if !exists {
				log.Printf("Warning: Permission %s not found", permissionName)
				continue
			}

			rolePermission := models.RolePermission{
				RoleID:       role.ID,
				PermissionID: permission.ID,
				GrantedAt:    time.Now(),
				GrantedBy:    adminID,
			}

			if err := db.Create(&rolePermission).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func seedPackages(db *gorm.DB) error {
	log.Println("Seeding packages...")

	packages := []models.Package{
		{
			ID:               uuid.New(),
			Name:             "Basis Beratung",
			Description:      "Grundlegende Elterngeld-Beratung für einfache Fälle",
			Type:             models.PackageTypeBasic,
			Price:            99.00,
			Currency:         "EUR",
			IsActive:         true,
			Features:         `["Erstberatung (30 Min)", "Grundlegende Antragsunterstützung", "E-Mail Support"]`,
			RequiresTimeslot: true,
			ManualAssignment: false,
			ConsultationTime: 30,
			HasFreePreTalk:   false,
			SortOrder:        1,
			BadgeText:        "Beliebt",
			BadgeColor:       "primary",
		},
		{
			ID:               uuid.New(),
			Name:             "Premium Beratung",
			Description:      "Umfassende Elterngeld-Beratung mit persönlicher Betreuung",
			Type:             models.PackageTypePremium,
			Price:            199.00,
			Currency:         "EUR",
			IsActive:         true,
			Features:         `["Ausführliche Beratung (60 Min)", "Vollständige Antragsbearbeitung", "Telefon & E-Mail Support", "1 Nachtermin inklusive"]`,
			RequiresTimeslot: true,
			ManualAssignment: false,
			ConsultationTime: 60,
			HasFreePreTalk:   true,
			PreTalkDuration:  15,
			SortOrder:        2,
			BadgeText:        "Empfohlen",
			BadgeColor:       "success",
		},
		{
			ID:               uuid.New(),
			Name:             "Komplett Service",
			Description:      "Rundum-sorglos-Paket mit vollständiger Betreuung",
			Type:             models.PackageTypeComplete,
			Price:            299.00,
			Currency:         "EUR",
			IsActive:         true,
			Features:         `["Umfassende Beratung (90 Min)", "Vollständige Antragsabwicklung", "Prioritäts-Support", "Unbegrenzte Nachfragen", "Dokumentenprüfung", "Behördenkommunikation"]`,
			RequiresTimeslot: true,
			ManualAssignment: true, // Requires manual assignment due to complexity
			ConsultationTime: 90,
			HasFreePreTalk:   true,
			PreTalkDuration:  15,
			SortOrder:        3,
			BadgeText:        "Premium",
			BadgeColor:       "warning",
		},
	}

	for i := range packages {
		packages[i].CreatedAt = time.Now()
		packages[i].UpdatedAt = time.Now()
	}

	return db.Create(&packages).Error
}

func seedAddons(db *gorm.DB) error {
	log.Println("Seeding addons...")

	addons := []models.Addon{
		{
			ID:          uuid.New(),
			Name:        "Expresszuschlag",
			Description: "Bearbeitung innerhalb von 24 Stunden",
			Price:       49.00,
			Currency:    "EUR",
			IsActive:    true,
			SortOrder:   1,
			Category:    "express",
		},
		{
			ID:          uuid.New(),
			Name:        "Zusätzlicher Nachtermin",
			Description: "Ein weiterer Beratungstermin (30 Min)",
			Price:       39.00,
			Currency:    "EUR",
			IsActive:    true,
			SortOrder:   2,
			Category:    "consultation",
		},
		{
			ID:          uuid.New(),
			Name:        "Dokumentenprüfung",
			Description: "Detaillierte Prüfung aller eingereichten Dokumente",
			Price:       29.00,
			Currency:    "EUR",
			IsActive:    true,
			SortOrder:   3,
			Category:    "documents",
		},
		{
			ID:          uuid.New(),
			Name:        "Einspruchsverfahren",
			Description: "Unterstützung bei Widerspruch gegen Elterngeldbescheid",
			Price:       89.00,
			Currency:    "EUR",
			IsActive:    true,
			SortOrder:   4,
			Category:    "legal",
		},
		{
			ID:          uuid.New(),
			Name:        "Steueroptimierung",
			Description: "Beratung zur steuerlichen Optimierung des Elterngeldes",
			Price:       69.00,
			Currency:    "EUR",
			IsActive:    true,
			SortOrder:   5,
			Category:    "tax",
		},
	}

	for i := range addons {
		addons[i].CreatedAt = time.Now()
		addons[i].UpdatedAt = time.Now()
	}

	return db.Create(&addons).Error
}

func seedPackageAddons(db *gorm.DB) error {
	log.Println("Seeding package-addon relationships...")

	// Get packages and addons
	var packages []models.Package
	if err := db.Find(&packages).Error; err != nil {
		return err
	}

	var addons []models.Addon
	if err := db.Find(&addons).Error; err != nil {
		return err
	}

	// Create maps for easy lookup
	packageMap := make(map[string]models.Package)
	for _, p := range packages {
		packageMap[p.Name] = p
	}

	addonMap := make(map[string]models.Addon)
	for _, a := range addons {
		addonMap[a.Name] = a
	}

	// Define package-addon relationships
	relationships := []struct {
		PackageName string
		AddonName   string
		IsDefault   bool
	}{
		{"Basis Beratung", "Expresszuschlag", false},
		{"Basis Beratung", "Zusätzlicher Nachtermin", false},
		{"Basis Beratung", "Dokumentenprüfung", false},

		{"Premium Beratung", "Expresszuschlag", false},
		{"Premium Beratung", "Dokumentenprüfung", true}, // Default for premium
		{"Premium Beratung", "Einspruchsverfahren", false},
		{"Premium Beratung", "Steueroptimierung", false},

		{"Komplett Service", "Dokumentenprüfung", true}, // Default for complete
		{"Komplett Service", "Einspruchsverfahren", true}, // Default for complete
		{"Komplett Service", "Steueroptimierung", false},
	}

	for _, rel := range relationships {
		pkg, pkgExists := packageMap[rel.PackageName]
		addon, addonExists := addonMap[rel.AddonName]

		if !pkgExists || !addonExists {
			continue
		}

		packageAddon := models.PackageAddon{
			PackageID: pkg.ID,
			AddonID:   addon.ID,
			IsDefault: rel.IsDefault,
			CreatedAt: time.Now(),
		}

		if err := db.Create(&packageAddon).Error; err != nil {
			return err
		}
	}

	return nil
}

func seedTimeslots(db *gorm.DB) error {
	log.Println("Seeding timeslots...")

	// Get berater users
	var beraters []models.User
	if err := db.Where("role IN ?", []models.UserRole{models.RoleBerater, models.RoleJuniorBerater}).Find(&beraters).Error; err != nil {
		return err
	}

	if len(beraters) == 0 {
		return nil
	}

	// Create timeslots for the next 30 days
	startDate := time.Now()
	endDate := startDate.Add(30 * 24 * time.Hour)

	var timeslots []models.Timeslot

	for current := startDate; current.Before(endDate); current = current.Add(24 * time.Hour) {
		// Skip weekends
		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			continue
		}

		// Create morning and afternoon slots for each berater
		for _, berater := range beraters {
			// Morning slots (9:00-12:00)
			morningSlots := []struct{ hour, minute int }{
				{9, 0}, {10, 0}, {11, 0},
			}

			for _, slot := range morningSlots {
				startTime := time.Date(current.Year(), current.Month(), current.Day(), slot.hour, slot.minute, 0, 0, current.Location())
				endTime := startTime.Add(60 * time.Minute)

				timeslot := models.Timeslot{
					ID:              uuid.New(),
					BeraterID:       berater.ID,
					Date:            current,
					StartTime:       startTime,
					EndTime:         endTime,
					Duration:        60,
					IsAvailable:     true,
					MaxBookings:     1,
					CurrentBookings: 0,
					Title:           "Beratungstermin",
					IsOnline:        true,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}

				timeslots = append(timeslots, timeslot)
			}

			// Afternoon slots (14:00-17:00)
			afternoonSlots := []struct{ hour, minute int }{
				{14, 0}, {15, 0}, {16, 0},
			}

			for _, slot := range afternoonSlots {
				startTime := time.Date(current.Year(), current.Month(), current.Day(), slot.hour, slot.minute, 0, 0, current.Location())
				endTime := startTime.Add(60 * time.Minute)

				timeslot := models.Timeslot{
					ID:              uuid.New(),
					BeraterID:       berater.ID,
					Date:            current,
					StartTime:       startTime,
					EndTime:         endTime,
					Duration:        60,
					IsAvailable:     true,
					MaxBookings:     1,
					CurrentBookings: 0,
					Title:           "Beratungstermin",
					IsOnline:        true,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}

				timeslots = append(timeslots, timeslot)
			}
		}
	}

	// Batch insert timeslots
	const batchSize = 100
	for i := 0; i < len(timeslots); i += batchSize {
		end := i + batchSize
		if end > len(timeslots) {
			end = len(timeslots)
		}

		if err := db.Create(timeslots[i:end]).Error; err != nil {
			return err
		}
	}

	return nil
}

func seedLeads(db *gorm.DB) error {
	log.Println("Seeding leads...")

	// Get users and beraters
	var users []models.User
	if err := db.Where("role = ?", models.RoleUser).Find(&users).Error; err != nil {
		return err
	}

	var beraters []models.User
	if err := db.Where("role IN ?", []models.UserRole{models.RoleBerater, models.RoleJuniorBerater}).Find(&beraters).Error; err != nil {
		return err
	}

	if len(users) == 0 || len(beraters) == 0 {
		return nil
	}

	leads := []models.Lead{
		{
			ID:                     uuid.New(),
			UserID:                 users[0].ID, // Lisa Schneider
			BeraterID:              &beraters[0].ID,
			Title:                  "Elterngeld für Zwillinge",
			Description:            "Beratung für Elterngeldantrag bei Zwillingsgeburt",
			Status:                 models.LeadStatusInProgress,
			Priority:               models.PriorityHigh,
			Source:                 models.LeadSourceWebsite,
			SourceDetails:          "Online-Antragsformular",
			ChildName:              "Emma und Liam",
			ChildBirthDate:         func() *time.Time { t := time.Now().Add(-30 * 24 * time.Hour); return &t }(),
			ExpectedAmount:         1800.00,
			ApplicationNumber:      "EG-2024-001234",
			PreferredContact:       "email",
			PreferredContactMethod: "email",
			IsQualified:            true,
			QualifiedAt:            func() *time.Time { t := time.Now().Add(-7 * 24 * time.Hour); return &t }(),
			EstimatedValue:         199.00,
			LeadScore:              85,
			LeadScoreReason:        "Hohe Wahrscheinlichkeit für Premium-Paket",
		},
		{
			ID:                     uuid.New(),
			UserID:                 users[1].ID, // Maria Weber (if exists)
			Title:                  "Erstberatung Elterngeld",
			Description:            "Grundlegende Beratung zum Elterngeldantrag",
			Status:                 models.LeadStatusNew,
			Priority:               models.PriorityMedium,
			Source:                 models.LeadSourceContact,
			SourceDetails:          "Kontaktformular Website",
			ChildName:              "Sophie",
			ChildBirthDate:         func() *time.Time { t := time.Now().Add(30 * 24 * time.Hour); return &t }(),
			ExpectedAmount:         1200.00,
			PreferredContact:       "phone",
			PreferredContactMethod: "phone",
			IsQualified:            false,
			EstimatedValue:         99.00,
			LeadScore:              65,
			NextFollowUpAt:         func() *time.Time { t := time.Now().Add(2 * 24 * time.Hour); return &t }(),
		},
		{
			ID:                     uuid.New(),
			UserID:                 users[2].ID, // Stefan Braun (if exists)
			BeraterID:              &beraters[0].ID,
			Title:                  "Elterngeld Plus Optimierung",
			Description:            "Optimierung der Elterngeld Plus Strategie",
			Status:                 models.LeadStatusCompleted,
			Priority:               models.PriorityMedium,
			Source:                 models.LeadSourceReferral,
			SourceDetails:          "Empfehlung von Freunden",
			ReferralSource:         "Familie Schmidt",
			ChildName:              "Max",
			ChildBirthDate:         func() *time.Time { t := time.Now().Add(-90 * 24 * time.Hour); return &t }(),
			ExpectedAmount:         1400.00,
			ApplicationNumber:      "EG-2024-001235",
			PreferredContact:       "both",
			PreferredContactMethod: "email",
			IsQualified:            true,
			QualifiedAt:            func() *time.Time { t := time.Now().Add(-14 * 24 * time.Hour); return &t }(),
			EstimatedValue:         299.00,
			ConvertedAt:            func() *time.Time { t := time.Now().Add(-7 * 24 * time.Hour); return &t }(),
			ConversionValue:        299.00,
			CompletedAt:            func() *time.Time { t := time.Now().Add(-1 * 24 * time.Hour); return &t }(),
			LeadScore:              95,
			LeadScoreReason:        "Hohe Zufriedenheit, potentieller Wiederholungskunde",
		},
	}

	for i := range leads {
		if i >= len(users) {
			break
		}
		leads[i].CreatedAt = time.Now().Add(-time.Duration(i*24) * time.Hour)
		leads[i].UpdatedAt = time.Now()
	}

	return db.Create(&leads).Error
}

// Continue with remaining seeders...
func seedBookings(db *gorm.DB) error {
	log.Println("Seeding bookings...")

	// Get users, packages, and leads
	var users []models.User
	if err := db.Where("role = ?", models.RoleUser).Find(&users).Error; err != nil {
		return err
	}

	var packages []models.Package
	if err := db.Find(&packages).Error; err != nil {
		return err
	}

	var leads []models.Lead
	if err := db.Limit(2).Find(&leads).Error; err != nil {
		return err
	}

	var beraters []models.User
	if err := db.Where("role IN ?", []models.UserRole{models.RoleBerater, models.RoleJuniorBerater}).Find(&beraters).Error; err != nil {
		return err
	}

	if len(users) == 0 || len(packages) == 0 || len(beraters) == 0 {
		return nil
	}

	bookings := []models.Booking{
		{
			ID:               uuid.New(),
			UserID:           users[0].ID,
			PackageID:        &packages[1].ID, // Premium package
			BeraterID:        &beraters[0].ID,
			LeadID:           func() *uuid.UUID { if len(leads) > 0 { return &leads[0].ID } else { return nil } }(),
			Title:            "Elterngeld Beratung - Premium",
			Description:      "Umfassende Beratung zum Elterngeldantrag",
			Type:             models.BookingTypeConsultation,
			Status:           models.BookingStatusConfirmed,
			ScheduledAt:      time.Now().Add(48 * time.Hour),
			Duration:         60,
			StartTime:        time.Now().Add(48 * time.Hour),
			EndTime:          time.Now().Add(48*time.Hour + 60*time.Minute),
			CustomerName:     "Lisa Schneider",
			CustomerEmail:    "user@example.com",
			CustomerPhone:    "+49 30 22222222",
			MeetingLink:      "https://meet.google.com/abc-defg-hij",
			IsOnline:         true,
			BookingReference: "BK-2024-ABC123",
			TotalAmount:      199.00,
			Currency:         "EUR",
			BookedAt:         time.Now().Add(-24 * time.Hour),
			ConfirmedAt:      func() *time.Time { t := time.Now().Add(-12 * time.Hour); return &t }(),
		},
		{
			ID:               uuid.New(),
			UserID:           users[0].ID,
			PackageID:        &packages[0].ID, // Basic package
			BeraterID:        &beraters[1].ID,
			Title:            "Vorgespräch - Kostenlos",
			Description:      "Kostenloses Vorgespräch zur Klärung der Situation",
			Type:             models.BookingTypePreTalk,
			Status:           models.BookingStatusCompleted,
			ScheduledAt:      time.Now().Add(-7 * 24 * time.Hour),
			Duration:         15,
			StartTime:        time.Now().Add(-7 * 24 * time.Hour),
			EndTime:          time.Now().Add(-7*24*time.Hour + 15*time.Minute),
			CustomerName:     "Lisa Schneider",
			CustomerEmail:    "user@example.com",
			CustomerPhone:    "+49 30 22222222",
			MeetingLink:      "https://meet.google.com/xyz-uvwx-rst",
			IsOnline:         true,
			BookingReference: "BK-2024-DEF456",
			TotalAmount:      0.00,
			Currency:         "EUR",
			BookedAt:         time.Now().Add(-10 * 24 * time.Hour),
			ConfirmedAt:      func() *time.Time { t := time.Now().Add(-9 * 24 * time.Hour); return &t }(),
			CompletedAt:      func() *time.Time { t := time.Now().Add(-7 * 24 * time.Hour); return &t }(),
		},
	}

	for i := range bookings {
		bookings[i].CreatedAt = time.Now().Add(-time.Duration(i*12) * time.Hour)
		bookings[i].UpdatedAt = time.Now()
	}

	return db.Create(&bookings).Error
}

func seedTodos(db *gorm.DB) error {
	log.Println("Seeding todos...")

	// Get users and bookings
	var users []models.User
	if err := db.Where("role = ?", models.RoleUser).Find(&users).Error; err != nil {
		return err
	}

	var beraters []models.User
	if err := db.Where("role IN ?", []models.UserRole{models.RoleBerater, models.RoleJuniorBerater}).Find(&beraters).Error; err != nil {
		return err
	}

	var bookings []models.Booking
	if err := db.Limit(2).Find(&bookings).Error; err != nil {
		return err
	}

	var leads []models.Lead
	if err := db.Limit(2).Find(&leads).Error; err != nil {
		return err
	}

	if len(users) == 0 || len(beraters) == 0 {
		return nil
	}

	todos := []models.Todo{
		{
			ID:          uuid.New(),
			BookingID:   func() *uuid.UUID { if len(bookings) > 0 { return &bookings[0].ID } else { return nil } }(),
			UserID:      users[0].ID,
			CreatedBy:   beraters[0].ID,
			Title:       "Gehaltsabrechnungen der letzten 12 Monate einreichen",
			Description: "Bitte reichen Sie alle Gehaltsabrechnungen der letzten 12 Monate ein, um das Elterngeld korrekt berechnen zu können.",
			IsCompleted: false,
			DueDate:     func() *time.Time { t := time.Now().Add(7 * 24 * time.Hour); return &t }(),
		},
		{
			ID:          uuid.New(),
			LeadID:      func() *uuid.UUID { if len(leads) > 0 { return &leads[0].ID } else { return nil } }(),
			UserID:      users[0].ID,
			CreatedBy:   beraters[0].ID,
			Title:       "Bescheinigung der Krankenkasse besorgen",
			Description: "Für den Elterngeldantrag benötigen wir eine Bescheinigung Ihrer Krankenkasse über die Mitgliedschaft.",
			IsCompleted: true,
			DueDate:     func() *time.Time { t := time.Now().Add(-2 * 24 * time.Hour); return &t }(),
			CompletedAt: func() *time.Time { t := time.Now().Add(-1 * 24 * time.Hour); return &t }(),
		},
		{
			ID:          uuid.New(),
			UserID:      users[0].ID,
			CreatedBy:   beraters[0].ID,
			Title:       "Mutterschaftsgeld-Bescheinigung einreichen",
			Description: "Falls Sie Mutterschaftsgeld erhalten haben, reichen Sie bitte die entsprechende Bescheinigung ein.",
			IsCompleted: false,
			DueDate:     func() *time.Time { t := time.Now().Add(14 * 24 * time.Hour); return &t }(),
		},
	}

	for i := range todos {
		todos[i].CreatedAt = time.Now().Add(-time.Duration(i*24) * time.Hour)
		todos[i].UpdatedAt = time.Now()
	}

	return db.Create(&todos).Error
}

func seedContactForms(db *gorm.DB) error {
	log.Println("Seeding contact forms...")

	contactForms := []models.ContactForm{
		{
			ID:         uuid.New(),
			Name:       "Sarah Müller",
			Email:      "sarah.mueller@example.com",
			Phone:      "+49 30 55555555",
			Subject:    "Frage zum Elterngeldantrag",
			Message:    "Hallo, ich erwarte mein erstes Kind und würde gerne wissen, wie ich den Elterngeldantrag am besten stelle. Können Sie mir dabei helfen?",
			Source:     "website",
			URL:        "https://elterngeld-portal.de/kontakt",
			UtmSource:  "google",
			UtmMedium:  "cpc",
			UtmCampaign: "elterngeld-beratung",
			IsProcessed: false,
			LeadCreated: false,
		},
		{
			ID:         uuid.New(),
			Name:       "Michael Weber",
			Email:      "michael.weber@example.com",
			Phone:      "+49 30 66666666",
			Subject:    "Terminanfrage für Beratung",
			Message:    "Guten Tag, ich möchte gerne einen Beratungstermin für Elterngeld Plus vereinbaren. Wann haben Sie die nächsten freien Termine?",
			Source:     "website",
			URL:        "https://elterngeld-portal.de/kontakt",
			IsProcessed: true,
			ProcessedAt: func() *time.Time { t := time.Now().Add(-12 * time.Hour); return &t }(),
			LeadCreated: true,
			IsReplied:   true,
			RepliedAt:   func() *time.Time { t := time.Now().Add(-6 * time.Hour); return &t }(),
		},
		{
			ID:         uuid.New(),
			Name:       "Anna Hoffmann",
			Email:      "anna.hoffmann@example.com",
			Subject:    "Frage zu den Preisen",
			Message:    "Hallo, können Sie mir die aktuellen Preise für Ihre Beratungsleistungen mitteilen? Gibt es auch Paketangebote?",
			Source:     "website",
			URL:        "https://elterngeld-portal.de/preise",
			IsProcessed: true,
			ProcessedAt: func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
			LeadCreated: false,
			IsReplied:   true,
			RepliedAt:   func() *time.Time { t := time.Now().Add(-18 * time.Hour); return &t }(),
		},
	}

	for i := range contactForms {
		contactForms[i].CreatedAt = time.Now().Add(-time.Duration((i+1)*24) * time.Hour)
		contactForms[i].UpdatedAt = time.Now()
	}

	return db.Create(&contactForms).Error
}

func seedJobs(db *gorm.DB) error {
	log.Println("Seeding jobs...")

	// Get admin user
	var adminUser models.User
	if err := db.Where("role = ?", models.RoleAdmin).First(&adminUser).Error; err != nil {
		return err
	}

	jobs := []models.Job{
		{
			ID:               uuid.New(),
			Title:            "Senior Elterngeld-Berater (m/w/d)",
			Slug:             "senior-elterngeld-berater-mwd",
			Description:      "Wir suchen einen erfahrenen Berater für die Betreuung unserer Premium-Kunden im Bereich Elterngeld und Familienleistungen.",
			ShortDescription: "Erfahrener Berater für Premium-Kunden gesucht",
			Status:           models.JobStatusPublished,
			Type:             models.JobTypeFullTime,
			Level:            models.JobLevelSenior,
			Department:       "Beratung",
			Location:         "Berlin",
			WorkLocation:     models.WorkLocationHybrid,
			IsRemote:         false,
			SalaryMin:        func() *float64 { v := 45000.0; return &v }(),
			SalaryMax:        func() *float64 { v := 60000.0; return &v }(),
			SalaryCurrency:   "EUR",
			SalaryPeriod:     "yearly",
			BenefitsText:     "30 Tage Urlaub, Homeoffice-Möglichkeit, Weiterbildungsbudget, betriebliche Altersvorsorge",
			RequiredSkills:   `["Beratungserfahrung", "Elterngeld-Kenntnisse", "Kundenbetreuung", "MS Office"]`,
			PreferredSkills:  `["Familienrecht", "Sozialversicherung", "CRM-Systeme"]`,
			RequiredExperience: "Mindestens 3 Jahre Erfahrung in der Sozialberatung oder ähnlichem Bereich",
			EducationRequired: "Abgeschlossenes Studium (BWL, Jura, Sozialwesen) oder vergleichbare Qualifikation",
			ContactEmail:     "jobs@elterngeld-portal.de",
			AllowDirectApply: true,
			Tags:             `["Vollzeit", "Berlin", "Beratung", "Elterngeld"]`,
			ViewCount:        45,
			ApplicationCount: 12,
			PublishedAt:      func() *time.Time { t := time.Now().Add(-10 * 24 * time.Hour); return &t }(),
			ExpiresAt:        func() *time.Time { t := time.Now().Add(20 * 24 * time.Hour); return &t }(),
			CreatedBy:        adminUser.ID,
		},
		{
			ID:               uuid.New(),
			Title:            "Junior Berater Elterngeld (m/w/d)",
			Slug:             "junior-berater-elterngeld-mwd",
			Description:      "Starten Sie Ihre Karriere in der Familienberatung! Wir bieten eine umfassende Einarbeitung und Weiterbildung.",
			ShortDescription: "Einstiegsposition für Berufseinsteiger",
			Status:           models.JobStatusPublished,
			Type:             models.JobTypeFullTime,
			Level:            models.JobLevelJunior,
			Department:       "Beratung",
			Location:         "Berlin / Remote",
			WorkLocation:     models.WorkLocationRemote,
			IsRemote:         true,
			SalaryMin:        func() *float64 { v := 32000.0; return &v }(),
			SalaryMax:        func() *float64 { v := 40000.0; return &v }(),
			SalaryCurrency:   "EUR",
			SalaryPeriod:     "yearly",
			BenefitsText:     "Flexible Arbeitszeiten, Vollzeit-Remote möglich, Mentoring-Programm",
			RequiredSkills:   `["Kommunikationsstärke", "Empathie", "Lernbereitschaft", "MS Office"]`,
			PreferredSkills:  `["Erste Beratungserfahrung", "Interesse an Familienthemen"]`,
			RequiredExperience: "Keine spezielle Berufserfahrung erforderlich - Quereinsteiger willkommen",
			EducationRequired: "Abgeschlossene Berufsausbildung oder Studium",
			ContactEmail:     "karriere@elterngeld-portal.de",
			AllowDirectApply: true,
			Tags:             `["Vollzeit", "Remote", "Berufseinsteiger", "Elterngeld"]`,
			ViewCount:        78,
			ApplicationCount: 23,
			PublishedAt:      func() *time.Time { t := time.Now().Add(-5 * 24 * time.Hour); return &t }(),
			ExpiresAt:        func() *time.Time { t := time.Now().Add(25 * 24 * time.Hour); return &t }(),
			CreatedBy:        adminUser.ID,
		},
		{
			ID:               uuid.New(),
			Title:            "Praktikant Marketing & Content (m/w/d)",
			Slug:             "praktikant-marketing-content-mwd",
			Description:      "Unterstützen Sie unser Marketing-Team bei der Erstellung von Content und der Durchführung von Kampagnen.",
			ShortDescription: "Praktikum im Marketing-Bereich",
			Status:           models.JobStatusDraft,
			Type:             models.JobTypeInternship,
			Level:            models.JobLevelEntry,
			Department:       "Marketing",
			Location:         "Berlin",
			WorkLocation:     models.WorkLocationOnSite,
			IsRemote:         false,
			SalaryMin:        func() *float64 { v := 800.0; return &v }(),
			SalaryCurrency:   "EUR",
			SalaryPeriod:     "monthly",
			BenefitsText:     "Praktikantenvergütung, flexible Arbeitszeiten, Übernahme-Möglichkeit",
			RequiredSkills:   `["Content-Erstellung", "Social Media", "Kreativität", "MS Office"]`,
			PreferredSkills:  `["Adobe Creative Suite", "WordPress", "SEO-Grundkenntnisse"]`,
			RequiredExperience: "Erste Erfahrungen im Marketing oder verwandten Bereichen von Vorteil",
			EducationRequired: "Laufendes Studium (Marketing, Kommunikation, BWL oder ähnlich)",
			ContactEmail:     "praktikum@elterngeld-portal.de",
			AllowDirectApply: true,
			Tags:             `["Praktikum", "Marketing", "Content", "Berlin"]`,
			ViewCount:        15,
			ApplicationCount: 3,
			CreatedBy:        adminUser.ID,
		},
	}

	for i := range jobs {
		jobs[i].CreatedAt = time.Now().Add(-time.Duration((i+1)*48) * time.Hour)
		jobs[i].UpdatedAt = time.Now()
	}

	return db.Create(&jobs).Error
}

func seedJobApplications(db *gorm.DB) error {
	log.Println("Seeding job applications...")

	// Get published jobs
	var jobs []models.Job
	if err := db.Where("status = ?", models.JobStatusPublished).Find(&jobs).Error; err != nil {
		return err
	}

	if len(jobs) == 0 {
		return nil
	}

	applications := []models.JobApplication{
		{
			ID:                jobs[0].ID, // Apply to senior position
			JobID:             jobs[0].ID,
			FirstName:         "Thomas",
			LastName:          "Becker",
			Email:             "thomas.becker@example.com",
			Phone:             "+49 30 77777777",
			Location:          "Berlin",
			Status:            models.ApplicationStatusReviewing,
			CoverLetter:       "Sehr geehrte Damen und Herren, hiermit bewerbe ich mich auf die Position als Senior Elterngeld-Berater. Mit meiner 5-jährigen Erfahrung in der Familienberatung bringe ich die notwendigen Qualifikationen mit...",
			YearsExperience:   5,
			CurrentPosition:   "Familienberater",
			CurrentCompany:    "Sozialberatung München GmbH",
			ExpectedSalary:    func() *float64 { v := 52000.0; return &v }(),
			AvailabilityDate:  func() *time.Time { t := time.Now().Add(30 * 24 * time.Hour); return &t }(),
			NoticePeriod:      "4 Wochen",
			MotivationText:    "Ich möchte Familien dabei helfen, ihre Ansprüche optimal geltend zu machen und dabei meine Expertise einbringen.",
			PrivacyConsent:    true,
			Source:            "website",
		},
		{
			ID:                uuid.New(),
			JobID:             jobs[1].ID, // Apply to junior position
			FirstName:         "Julia",
			LastName:          "Wagner",
			Email:             "julia.wagner@example.com",
			Phone:             "+49 30 88888888",
			Location:          "Hamburg",
			Status:            models.ApplicationStatusSubmitted,
			CoverLetter:       "Liebe Personalabteilung, als Quereinsteigerin aus dem Sozialwesen interessiere ich mich sehr für die Position als Junior Beraterin...",
			YearsExperience:   2,
			CurrentPosition:   "Sozialarbeiterin",
			CurrentCompany:    "Jugendamt Hamburg",
			ExpectedSalary:    func() *float64 { v := 36000.0; return &v }(),
			AvailabilityDate:  func() *time.Time { t := time.Now().Add(60 * 24 * time.Hour); return &t }(),
			NoticePeriod:      "6 Wochen",
			MotivationText:    "Die Arbeit mit Familien liegt mir sehr am Herzen und ich möchte mich speziell im Bereich Elterngeld weiterbilden.",
			PrivacyConsent:    true,
			NewsletterConsent: true,
			Source:            "linkedin",
			SourceDetails:     "LinkedIn Job Post",
		},
		{
			ID:                uuid.New(),
			JobID:             jobs[1].ID, // Another application to junior position
			FirstName:         "Mark",
			LastName:          "Fischer",
			Email:             "mark.fischer@example.com",
			Phone:             "+49 30 99999999",
			Location:          "Berlin",
			Status:            models.ApplicationStatusInterview,
			CoverLetter:       "Sehr geehrtes Team, mit großem Interesse bewerbe ich mich auf die ausgeschriebene Position...",
			YearsExperience:   1,
			CurrentPosition:   "Kundenberater",
			CurrentCompany:    "Versicherung AG",
			ExpectedSalary:    func() *float64 { v := 38000.0; return &v }(),
			AvailabilityDate:  func() *time.Time { t := time.Now().Add(45 * 24 * time.Hour); return &t }(),
			NoticePeriod:      "4 Wochen",
			MotivationText:    "Ich suche eine neue Herausforderung in einem sinnstiftenden Bereich.",
			PrivacyConsent:    true,
			Source:            "website",
			InterviewScheduled: true,
			InterviewDate:     func() *time.Time { t := time.Now().Add(7 * 24 * time.Hour); return &t }(),
		},
	}

	for i := range applications {
		if i == 0 {
			applications[i].ID = uuid.New() // Fix the first entry
		}
		applications[i].CreatedAt = time.Now().Add(-time.Duration((i+1)*24) * time.Hour)
		applications[i].UpdatedAt = time.Now()
	}

	return db.Create(&applications).Error
}

func seedNotificationPreferences(db *gorm.DB) error {
	log.Println("Seeding notification preferences...")

	// Get all users
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	var preferences []models.NotificationPreference

	for _, user := range users {
		preference := models.NotificationPreference{
			ID:                            uuid.New(),
			UserID:                        user.ID,
			EmailEnabled:                  true,
			EmailBookingNotifications:     true,
			EmailPaymentNotifications:     true,
			EmailMarketingNotifications:   user.Role == models.RoleUser, // Only users get marketing by default
			EmailTodoNotifications:        true,
			EmailReminderNotifications:    true,
			SMSEnabled:                    false,
			SMSBookingNotifications:       false,
			SMSReminderNotifications:      false,
			InAppEnabled:                  true,
			InAppBookingNotifications:     true,
			InAppTodoNotifications:        true,
			PushEnabled:                   false,
			PushBookingNotifications:      false,
			PushReminderNotifications:     false,
			QuietHoursEnabled:             false,
			Timezone:                      "Europe/Berlin",
			CreatedAt:                     time.Now(),
			UpdatedAt:                     time.Now(),
		}

		preferences = append(preferences, preference)
	}

	return db.Create(&preferences).Error
}