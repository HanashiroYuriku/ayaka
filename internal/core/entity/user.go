package entity

import "time"

// Role is the authorization role of a user.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) String() string { return string(r) }

// User is the persistent user aggregate.
//
// NOTE: the database schema is owned by the SQL migrations in /migrations.
// The gorm tags below only describe the mapping (and are used by the
// in-memory SQLite test database); keep both in sync.
type User struct {
	BaseEntity
	Username        string     `gorm:"type:varchar(20);not null;uniqueIndex"`
	Email           string     `gorm:"type:varchar(100);not null;uniqueIndex"`
	DisplayName     string     `gorm:"type:varchar(100)"`
	Password        string     `gorm:"type:text;not null"`
	Role            Role       `gorm:"type:varchar(20);not null;default:user"`
	EmailVerifiedAt *time.Time `gorm:""`

	// Brute-force protection: reset on successful login / password reset.
	FailedLoginAttempts int        `gorm:"not null;default:0"`
	LockedUntil         *time.Time `gorm:""`
}

func (User) TableName() string { return "users" }

// IsEmailVerified reports whether the user confirmed their email address.
func (u *User) IsEmailVerified() bool { return u.EmailVerifiedAt != nil }

// IsLocked reports whether login is temporarily blocked at time now.
func (u *User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// DisplayNameOrUsername is the friendliest name to address the user with.
func (u *User) DisplayNameOrUsername() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}
