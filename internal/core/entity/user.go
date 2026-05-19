package entity

type User struct {
	BaseEntity
	Name string `json:"name"`
	Email string `json:"email" gorm:"unique"`
	Password string `json:"-"`
	Role string `json:"role"`
}