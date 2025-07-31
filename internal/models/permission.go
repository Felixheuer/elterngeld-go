package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionAction string

const (
	PermissionActionCreate PermissionAction = "create"
	PermissionActionRead   PermissionAction = "read"
	PermissionActionUpdate PermissionAction = "update"
	PermissionActionDelete PermissionAction = "delete"
	PermissionActionList   PermissionAction = "list"
	PermissionActionManage PermissionAction = "manage" // full access
)

type PermissionResource string

const (
	// Dashboard permissions
	PermissionResourceDashboard     PermissionResource = "dashboard"
	PermissionResourceAdminDashboard PermissionResource = "dashboard.admin"
	PermissionResourceUserDashboard  PermissionResource = "dashboard.user"
	
	// User management
	PermissionResourceUser     PermissionResource = "user"
	PermissionResourceUsers    PermissionResource = "users"
	PermissionResourceProfile  PermissionResource = "profile"
	
	// Lead management
	PermissionResourceLead      PermissionResource = "lead"
	PermissionResourceLeads     PermissionResource = "leads"
	PermissionResourceOwnLeads  PermissionResource = "leads.own"
	PermissionResourceAllLeads  PermissionResource = "leads.all"
	
	// Booking management
	PermissionResourceBooking     PermissionResource = "booking"
	PermissionResourceBookings    PermissionResource = "bookings"
	PermissionResourceOwnBookings PermissionResource = "bookings.own"
	PermissionResourceAllBookings PermissionResource = "bookings.all"
	
	// Package management
	PermissionResourcePackage  PermissionResource = "package"
	PermissionResourcePackages PermissionResource = "packages"
	PermissionResourceAddon    PermissionResource = "addon"
	PermissionResourceAddons   PermissionResource = "addons"
	
	// Payment management
	PermissionResourcePayment  PermissionResource = "payment"
	PermissionResourcePayments PermissionResource = "payments"
	
	// Calendar and timeslot management
	PermissionResourceCalendar   PermissionResource = "calendar"
	PermissionResourceTimeslot   PermissionResource = "timeslot"
	PermissionResourceTimeslots  PermissionResource = "timeslots"
	PermissionResourceOwnCalendar PermissionResource = "calendar.own"
	PermissionResourceAllCalendars PermissionResource = "calendar.all"
	
	// Todo management
	PermissionResourceTodo      PermissionResource = "todo"
	PermissionResourceTodos     PermissionResource = "todos"
	PermissionResourceOwnTodos  PermissionResource = "todos.own"
	PermissionResourceAllTodos  PermissionResource = "todos.all"
	
	// Document management
	PermissionResourceDocument  PermissionResource = "document"
	PermissionResourceDocuments PermissionResource = "documents"
	
	// Settings and configuration
	PermissionResourceSettings        PermissionResource = "settings"
	PermissionResourceSystemSettings  PermissionResource = "settings.system"
	PermissionResourceSecuritySettings PermissionResource = "settings.security"
	
	// Job management
	PermissionResourceJob  PermissionResource = "job"
	PermissionResourceJobs PermissionResource = "jobs"
	
	// Contact forms
	PermissionResourceContactForm  PermissionResource = "contact_form"
	PermissionResourceContactForms PermissionResource = "contact_forms"
	
	// Reports and analytics
	PermissionResourceReport   PermissionResource = "report"
	PermissionResourceReports  PermissionResource = "reports"
	PermissionResourceAnalytics PermissionResource = "analytics"
	
	// Communication
	PermissionResourceEmail         PermissionResource = "email"
	PermissionResourceNotification  PermissionResource = "notification"
	PermissionResourceNotifications PermissionResource = "notifications"
)

// Permission represents a specific permission in the system
type Permission struct {
	ID          uuid.UUID          `json:"id" gorm:"type:char(36);primary_key"`
	Name        string             `json:"name" gorm:"not null;uniqueIndex"`
	Resource    PermissionResource `json:"resource" gorm:"not null"`
	Action      PermissionAction   `json:"action" gorm:"not null"`
	Description string             `json:"description" gorm:"type:text"`
	IsActive    bool               `json:"is_active" gorm:"not null;default:true"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Roles []Role `json:"roles,omitempty" gorm:"many2many:role_permissions;"`
}

// Role represents a role that can be assigned to users
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex"`
	DisplayName string    `json:"display_name" gorm:"not null"`
	Description string    `json:"description" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"not null;default:true"`
	IsDefault   bool      `json:"is_default" gorm:"not null;default:false"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	Users       []User       `json:"users,omitempty" gorm:"many2many:user_roles;"`
}

// RolePermission represents the junction table for role-permission relationships
type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" gorm:"type:char(36);primary_key"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:char(36);primary_key"`
	GrantedAt    time.Time `json:"granted_at" gorm:"not null"`
	GrantedBy    uuid.UUID `json:"granted_by" gorm:"type:char(36);not null"`
	
	// Relationships
	Role       Role       `json:"role,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Permission Permission `json:"permission,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Granter    User       `json:"granter,omitempty" gorm:"foreignKey:GrantedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// UserRole represents user-role assignments
type UserRole struct {
	UserID     uuid.UUID `json:"user_id" gorm:"type:char(36);primary_key"`
	RoleID     uuid.UUID `json:"role_id" gorm:"type:char(36);primary_key"`
	AssignedAt time.Time `json:"assigned_at" gorm:"not null"`
	AssignedBy uuid.UUID `json:"assigned_by" gorm:"type:char(36);not null"`
	ExpiresAt  *time.Time `json:"expires_at" gorm:""`
	IsActive   bool       `json:"is_active" gorm:"not null;default:true"`
	
	// Relationships
	User     User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Role     Role `json:"role,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Assigner User `json:"assigner,omitempty" gorm:"foreignKey:AssignedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// UserPermission represents direct user permissions (override role permissions)
type UserPermission struct {
	ID           uuid.UUID        `json:"id" gorm:"type:char(36);primary_key"`
	UserID       uuid.UUID        `json:"user_id" gorm:"type:char(36);not null;index"`
	PermissionID uuid.UUID        `json:"permission_id" gorm:"type:char(36);not null;index"`
	IsGranted    bool             `json:"is_granted" gorm:"not null;default:true"` // false = explicitly denied
	GrantedAt    time.Time        `json:"granted_at" gorm:"not null"`
	GrantedBy    uuid.UUID        `json:"granted_by" gorm:"type:char(36);not null"`
	ExpiresAt    *time.Time       `json:"expires_at" gorm:""`
	Reason       string           `json:"reason" gorm:"type:text"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	User       User       `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Permission Permission `json:"permission,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Granter    User       `json:"granter,omitempty" gorm:"foreignKey:GrantedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// PermissionTemplate represents predefined permission sets for quick role creation
type PermissionTemplate struct {
	ID          uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex"`
	DisplayName string    `json:"display_name" gorm:"not null"`
	Description string    `json:"description" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"not null;default:true"`
	
	// Permission data (JSON array of permission names)
	PermissionNames string `json:"permission_names" gorm:"type:text;not null"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Response DTOs
type PermissionResponse struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	Resource    PermissionResource `json:"resource"`
	Action      PermissionAction   `json:"action"`
	Description string             `json:"description"`
	IsActive    bool               `json:"is_active"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type RoleResponse struct {
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	DisplayName    string               `json:"display_name"`
	Description    string               `json:"description"`
	IsActive       bool                 `json:"is_active"`
	IsDefault      bool                 `json:"is_default"`
	SortOrder      int                  `json:"sort_order"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	Permissions    []PermissionResponse `json:"permissions,omitempty"`
	PermissionCount int                 `json:"permission_count"`
	UserCount      int                  `json:"user_count"`
}

type UserPermissionResponse struct {
	ID           uuid.UUID          `json:"id"`
	UserID       uuid.UUID          `json:"user_id"`
	Permission   PermissionResponse `json:"permission"`
	IsGranted    bool               `json:"is_granted"`
	GrantedAt    time.Time          `json:"granted_at"`
	ExpiresAt    *time.Time         `json:"expires_at"`
	Reason       string             `json:"reason"`
	GrantedBy    *UserResponse      `json:"granted_by,omitempty"`
	IsExpired    bool               `json:"is_expired"`
}

// Request DTOs
type CreatePermissionRequest struct {
	Name        string             `json:"name" validate:"required"`
	Resource    PermissionResource `json:"resource" validate:"required"`
	Action      PermissionAction   `json:"action" validate:"required,oneof=create read update delete list manage"`
	Description string             `json:"description"`
}

type UpdatePermissionRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type CreateRoleRequest struct {
	Name           string      `json:"name" validate:"required"`
	DisplayName    string      `json:"display_name" validate:"required"`
	Description    string      `json:"description"`
	PermissionIDs  []uuid.UUID `json:"permission_ids"`
	PermissionNames []string   `json:"permission_names"`
	SortOrder      int         `json:"sort_order"`
}

type UpdateRoleRequest struct {
	Name           *string     `json:"name"`
	DisplayName    *string     `json:"display_name"`
	Description    *string     `json:"description"`
	PermissionIDs  []uuid.UUID `json:"permission_ids"`
	PermissionNames []string   `json:"permission_names"`
	SortOrder      *int        `json:"sort_order"`
	IsActive       *bool       `json:"is_active"`
}

type AssignRoleRequest struct {
	UserID    uuid.UUID  `json:"user_id" validate:"required"`
	RoleID    uuid.UUID  `json:"role_id" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type GrantPermissionRequest struct {
	UserID       uuid.UUID  `json:"user_id" validate:"required"`
	PermissionID uuid.UUID  `json:"permission_id" validate:"required"`
	IsGranted    bool       `json:"is_granted"`
	ExpiresAt    *time.Time `json:"expires_at"`
	Reason       string     `json:"reason"`
}

type CheckPermissionRequest struct {
	UserID   uuid.UUID          `json:"user_id" validate:"required"`
	Resource PermissionResource `json:"resource" validate:"required"`
	Action   PermissionAction   `json:"action" validate:"required,oneof=create read update delete list manage"`
}

// BeforeCreate hooks
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Name == "" {
		p.Name = string(p.Resource) + "." + string(p.Action)
	}
	return nil
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (up *UserPermission) BeforeCreate(tx *gorm.DB) error {
	if up.ID == uuid.Nil {
		up.ID = uuid.New()
	}
	return nil
}

func (pt *PermissionTemplate) BeforeCreate(tx *gorm.DB) error {
	if pt.ID == uuid.Nil {
		pt.ID = uuid.New()
	}
	return nil
}

// Helper methods
func (p *Permission) ToResponse() PermissionResponse {
	return PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func (r *Role) ToResponse() RoleResponse {
	response := RoleResponse{
		ID:              r.ID,
		Name:            r.Name,
		DisplayName:     r.DisplayName,
		Description:     r.Description,
		IsActive:        r.IsActive,
		IsDefault:       r.IsDefault,
		SortOrder:       r.SortOrder,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		PermissionCount: len(r.Permissions),
		UserCount:       len(r.Users),
	}
	
	// Convert permissions
	for _, permission := range r.Permissions {
		response.Permissions = append(response.Permissions, permission.ToResponse())
	}
	
	return response
}

func (up *UserPermission) ToResponse() UserPermissionResponse {
	response := UserPermissionResponse{
		ID:        up.ID,
		UserID:    up.UserID,
		Permission: up.Permission.ToResponse(),
		IsGranted: up.IsGranted,
		GrantedAt: up.GrantedAt,
		ExpiresAt: up.ExpiresAt,
		Reason:    up.Reason,
		IsExpired: up.IsExpired(),
	}
	
	if up.Granter.ID != uuid.Nil {
		granterResponse := up.Granter.ToResponse()
		response.GrantedBy = &granterResponse
	}
	
	return response
}

func (up *UserPermission) IsExpired() bool {
	if up.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*up.ExpiresAt)
}

func (ur *UserRole) IsExpired() bool {
	if ur.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ur.ExpiresAt)
}

func (ur *UserRole) IsActiveAndValid() bool {
	return ur.IsActive && !ur.IsExpired()
}

// Permission checker methods
func (p *Permission) Matches(resource PermissionResource, action PermissionAction) bool {
	// Exact match
	if p.Resource == resource && p.Action == action {
		return true
	}
	
	// Manage permission grants all actions
	if p.Resource == resource && p.Action == PermissionActionManage {
		return true
	}
	
	// Check hierarchical permissions (e.g., "users" includes "user")
	if p.Action == action || p.Action == PermissionActionManage {
		return p.matchesResource(resource)
	}
	
	return false
}

func (p *Permission) matchesResource(resource PermissionResource) bool {
	resourceStr := string(p.Resource)
	targetStr := string(resource)
	
	// Exact match
	if resourceStr == targetStr {
		return true
	}
	
	// Check if resource is a parent of target
	// e.g., "users" matches "user", "leads.all" matches "leads.own"
	if strings.HasPrefix(targetStr, resourceStr+".") {
		return true
	}
	
	// Check wildcard patterns
	if strings.HasSuffix(resourceStr, ".all") {
		baseResource := strings.TrimSuffix(resourceStr, ".all")
		return strings.HasPrefix(targetStr, baseResource)
	}
	
	return false
}

// Helper functions for common permission patterns
func CreatePermissionName(resource PermissionResource, action PermissionAction) string {
	return string(resource) + "." + string(action)
}

func ParsePermissionName(name string) (PermissionResource, PermissionAction, error) {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid permission name format: %s", name)
	}
	
	action := PermissionAction(parts[len(parts)-1])
	resource := PermissionResource(strings.Join(parts[:len(parts)-1], "."))
	
	return resource, action, nil
}

// Common permission sets for easy role creation
var DefaultPermissions = map[UserRole][]string{
	"user": {
		"dashboard.user.read",
		"profile.read",
		"profile.update",
		"bookings.own.read",
		"bookings.own.create",
		"bookings.own.update",
		"todos.own.read",
		"todos.own.update",
		"documents.own.read",
		"documents.own.create",
		"packages.read",
		"contact_form.create",
	},
	"berater": {
		"dashboard.read",
		"leads.read",
		"leads.create",
		"leads.update",
		"leads.own.manage",
		"bookings.read",
		"bookings.create",
		"bookings.update",
		"calendar.own.manage",
		"timeslots.own.manage",
		"todos.create",
		"todos.update",
		"todos.read",
		"documents.read",
		"documents.create",
		"contact_forms.read",
		"contact_forms.update",
		"users.read",
		"packages.read",
		"profile.read",
		"profile.update",
	},
	"junior_berater": {
		"dashboard.read",
		"leads.read",
		"leads.own.update",
		"bookings.read",
		"calendar.own.read",
		"timeslots.own.read",
		"todos.read",
		"documents.read",
		"contact_forms.read",
		"users.read",
		"packages.read",
		"profile.read",
		"profile.update",
	},
	"admin": {
		// Admin gets all permissions - this would be populated dynamically
		"*.manage",
	},
}