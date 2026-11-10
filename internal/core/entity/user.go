package entity

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
	Username    string `gorm:"type:varchar(20);not null;uniqueIndex"`
	Email       string `gorm:"type:varchar(100);not null;uniqueIndex"`
	DisplayName string `gorm:"type:varchar(100)"`
	Password    string `gorm:"type:text;not null"`
	Role        Role   `gorm:"type:varchar(20);not null;default:user"`
}

func (User) TableName() string { return "users" }
