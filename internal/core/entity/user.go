package entity

type User struct {
	BaseEntity
	Username string `json:"username"`
	Email    string `json:"email" gorm:"unique"`
	Password string `json:"-"`
	Role     string `json:"role"`
}
