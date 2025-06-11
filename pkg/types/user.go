package types

import "time"


type User struct {
	ID        string `json:"id"`
	UserName  string `json:"user_name"`
	Email     string `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}


type CreateUserRequest struct {
	UserName string `json:"user_name"`
	Email 	 string `json:"email"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	User User `json:"user"`
	Token string `json:"token"`
}



type GetUserResponse struct {
	User User `json:"user"`
}