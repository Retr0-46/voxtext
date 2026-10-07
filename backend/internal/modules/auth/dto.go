package auth

import "time"

type registerRequest struct {
	Username    string `json:"username"`
	Usersurname string `json:"usersurname"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Usersurname string    `json:"usersurname"`
	Email       string    `json:"email"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type LoginResponse struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}
