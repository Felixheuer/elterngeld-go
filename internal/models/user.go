package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser         UserRole = "user"
	RoleBerater      UserRole = "berater"
	RoleJuniorBerater UserRole = "junior_berater"
	RoleAdmin        UserRole = "admin"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Password  string    `json:"-" gorm:"not null" validate:"required,min=6"`
	FirstName string    `json:"first_name" gorm:"not null" validate:"required"`
	LastName  string    `json:"last_name" gorm:"not null" validate:"required"`
	Phone     string    `json:"phone" gorm:""`
	Role      UserRole  `json:"role" gorm:"not null;default:'user'" validate:"required,oneof=user berater junior_berater admin"`
	IsActive  bool      `json:"is_active" gorm:"not null;default:true"`

	// Profile information
	DateOfBirth *time.Time `json:"date_of_birth" gorm:""`
	Address     string     `json:"address" gorm:""`
	PostalCode  string     `json:"postal_code" gorm:""`
	City        string     `json:"city" gorm:""`

	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Email verification
	EmailVerified   bool       `json:"email_verified" gorm:"not null;default:false"`
	EmailVerifiedAt *time.Time `json:"email_verified_at" gorm:""`

	// Password reset
	ResetToken    string     `json:"-" gorm:""`
	ResetTokenExp *time.Time `json:"-" gorm:""`

	// Relationships
	Leads         []Lead         `json:"leads,omitempty" gorm:"foreignKey:UserID"`
	AssignedLeads []Lead         `json:"assigned_leads,omitempty" gorm:"foreignKey:BeraterID"`
	Activities    []Activity     `json:"activities,omitempty" gorm:"foreignKey:UserID"`
	RefreshTokens []RefreshToken `json:"-" gorm:"foreignKey:UserID"`
	
	// New relationships for booking system
	Bookings      []Booking      `json:"bookings,omitempty" gorm:"foreignKey:UserID"`
	BeraterBookings []Booking    `json:"berater_bookings,omitempty" gorm:"foreignKey:BeraterID"`
	Timeslots     []Timeslot     `json:"timeslots,omitempty" gorm:"foreignKey:BeraterID"`
	AssignedTodos []Todo         `json:"assigned_todos,omitempty" gorm:"foreignKey:UserID"`
	CreatedTodos  []Todo         `json:"created_todos,omitempty" gorm:"foreignKey:CreatedBy"`
	
	// Notification relationships
	Notifications []Notification `json:"notifications,omitempty" gorm:"foreignKey:UserID"`
	EmailVerifications []EmailVerification `json:"-" gorm:"foreignKey:UserID"`
	PasswordResets []PasswordReset `json:"-" gorm:"foreignKey:UserID"`
	NotificationPreferences *NotificationPreference `json:"notification_preferences,omitempty" gorm:"foreignKey:UserID"`
	
	// Permission relationships
	Roles         []Role         `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	UserPermissions []UserPermission `json:"user_permissions,omitempty" gorm:"foreignKey:UserID"`
	
	// Job relationships
	CreatedJobs   []Job          `json:"created_jobs,omitempty" gorm:"foreignKey:CreatedBy"`
	ReviewedApplications []JobApplication `json:"reviewed_applications,omitempty" gorm:"foreignKey:ReviewedBy"`
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	Token     string    `json:"-" gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	IsRevoked bool      `json:"is_revoked" gorm:"not null;default:false"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null"`

	// Relationships
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// UserResponse represents the user data returned in API responses (without sensitive data)
type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Phone         string     `json:"phone"`
	Role          UserRole   `json:"role"`
	IsActive      bool       `json:"is_active"`
	DateOfBirth   *time.Time `json:"date_of_birth"`
	Address       string     `json:"address"`
	PostalCode    string     `json:"postal_code"`
	City          string     `json:"city"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Password  string   `json:"password" validate:"required,min=6"`
	FirstName string   `json:"first_name" validate:"required"`
	LastName  string   `json:"last_name" validate:"required"`
	Phone     string   `json:"phone"`
	Role      UserRole `json:"role" validate:"omitempty,oneof=user berater junior_berater admin"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	FirstName   *string    `json:"first_name"`
	LastName    *string    `json:"last_name"`
	Phone       *string    `json:"phone"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Address     *string    `json:"address"`
	PostalCode  *string    `json:"postal_code"`
	City        *string    `json:"city"`
}

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

// BeforeCreate is a GORM hook that runs before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return u.HashPassword()
}

// HashPassword hashes the user's password
func (u *User) HashPassword() error {
	if u.Password == "" {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword checks if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Phone:         u.Phone,
		Role:          u.Role,
		IsActive:      u.IsActive,
		DateOfBirth:   u.DateOfBirth,
		Address:       u.Address,
		PostalCode:    u.PostalCode,
		City:          u.City,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsBerater checks if the user has berater role
func (u *User) IsBerater() bool {
	return u.Role == RoleBerater
}

// IsUser checks if the user has user role
func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

// IsJuniorBerater checks if the user has junior_berater role
func (u *User) IsJuniorBerater() bool {
	return u.Role == RoleJuniorBerater
}
