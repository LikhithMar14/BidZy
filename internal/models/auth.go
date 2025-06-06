package models

type SignUpRequest struct {
    UserName        string `json:"userName" validate:"required,min=1,max=50"`
    Email           string `json:"email" validate:"required,email"`
    Password        string `json:"password" validate:"required,min=1"` 
    ConfirmPassword string `json:"confirmPassword" validate:"required,eqfield=Password"`
}

type SignInRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8,max=50"`
}