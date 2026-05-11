// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model

import (
	"fmt"
	"time"
)

// UserStatus represents the account lifecycle state of a user.
type UserStatus string

// User account lifecycle states.
const (
	UserStatusPending   UserStatus = "pending"
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
	UserStatusDeleted   UserStatus = "deleted"
)

// Role represents the access level of a user within the system.
type Role int

// Role levels — higher value means broader access.
const (
	RoleUser       Role = 10
	RoleInstructor Role = 20
	RoleModerator  Role = 30
	RoleManager    Role = 40
	RoleAdmin      Role = 50
	RoleSuperAdmin Role = 60
	RoleSystem     Role = 100
)

// String returns the string representation of a Role.
func (r Role) String() string {
	switch r {
	case RoleUser:
		return "user"
	case RoleInstructor:
		return "instructor"
	case RoleModerator:
		return "moderator"
	case RoleManager:
		return "manager"
	case RoleAdmin:
		return "admin"
	case RoleSuperAdmin:
		return "superadmin"
	case RoleSystem:
		return "system"
	default:
		return "user"
	}
}

// ParseRole converts a string to a Role.
func ParseRole(s string) Role {
	switch s {
	case "instructor":
		return RoleInstructor
	case "moderator":
		return RoleModerator
	case "manager":
		return RoleManager
	case "admin":
		return RoleAdmin
	case "superadmin":
		return RoleSuperAdmin
	case "system":
		return RoleSystem
	default:
		return RoleUser
	}
}

// Scan implements sql.Scanner so Role can be read directly from a database ENUM column.
func (r *Role) Scan(val any) error {
	if val == nil {
		*r = RoleUser
		return nil
	}
	switch v := val.(type) {
	case []byte:
		*r = ParseRole(string(v))
	case string:
		*r = ParseRole(v)
	default:
		return fmt.Errorf("unsupported role type: %T", val)
	}
	return nil
}

// User represents a registered account.
type User struct {
	// Identity
	ID           string     `db:"id"             json:"id"`
	Email        string     `db:"email"          json:"email"`
	PasswordHash string     `db:"password_hash"  json:"-"`
	Status       UserStatus `db:"status"         json:"status"`
	IsVerified   bool       `db:"is_verified"    json:"is_verified"`
	Is2FAEnabled bool       `db:"is_2fa_enabled" json:"is_2fa_enabled"`
	Role         Role       `db:"role"           json:"role"`

	// Profile
	FirstName   string  `db:"first_name"   json:"first_name"`
	LastName    string  `db:"last_name"    json:"last_name"`
	CompanyName *string `db:"company_name" json:"company_name,omitempty"`
	Headline    *string `db:"headline"     json:"headline,omitempty"`
	Bio         *string `db:"bio"          json:"bio,omitempty"`
	AvatarURL   *string `db:"avatar_url"   json:"avatar_url,omitempty"`
	Website     *string `db:"website"      json:"website,omitempty"`

	// Address
	AddressLine1 *string  `db:"address_line1" json:"address_line1,omitempty"`
	AddressLine2 *string  `db:"address_line2" json:"address_line2,omitempty"`
	City         *string  `db:"city"          json:"city,omitempty"`
	PostalCode   *string  `db:"postal_code"   json:"postal_code,omitempty"`
	Latitude     *float64 `db:"latitude"      json:"latitude,omitempty"`
	Longitude    *float64 `db:"longitude"     json:"longitude,omitempty"`

	// Business
	VATNumber   *string `db:"vat_number"   json:"-"` // encrypted — never expose raw
	CountryCode *string `db:"country_code" json:"country_code,omitempty"`

	// Locale
	Language  string     `db:"language"   json:"language"`
	Timezone  string     `db:"timezone"   json:"timezone"`
	BirthDate *time.Time `db:"birth_date" json:"-"` // encrypted — never expose raw

	// Activity
	LastLoginAt         *time.Time `db:"last_login_at"          json:"last_login_at,omitempty"`
	LastLoginIP         *string    `db:"last_login_ip"          json:"-"` // encrypted — never expose raw
	LoginCount          uint       `db:"login_count"            json:"login_count"`
	FailedLoginAttempts int        `db:"failed_login_attempts"  json:"-"`
	LockedUntil         *time.Time `db:"locked_until"           json:"-"`

	// Timestamps
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	BannedAt  *time.Time `db:"banned_at"  json:"banned_at,omitempty"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"` // soft delete — internal only
}
