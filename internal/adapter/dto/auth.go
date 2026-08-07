package dto

// LoginRequest authenticates an existing admin.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest creates an admin account, used by both /auth/setup and
// /admin/users.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse carries the signed JWT.
type LoginResponse struct {
	Token string `json:"token"`
}

// CreatedAdminResponse describes a newly created admin account.
type CreatedAdminResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
