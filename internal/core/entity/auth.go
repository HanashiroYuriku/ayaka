package entity

type UserRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=20,unique=users->username,whitespace,username" example:"hanashiroyuriku"`
	Email       string `json:"email" validate:"required,max=100,email,unique=users->email,whitespace" example:"hanashiroyuriku@gmail.com"`
	DisplayName string `json:"displayName" example:"Hanashiro Yuriku"`
	Password    string `json:"password" validate:"required,complexpassword" example:"P4$$word"`
}
