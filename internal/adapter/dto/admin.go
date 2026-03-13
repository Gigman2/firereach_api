package dto

import "github.com/firereach/api/internal/domain"

type AdminUserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func ToAdminUserResponse(u domain.AdminUser) AdminUserResponse {
	return AdminUserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func ToAdminUserListResponse(users []domain.AdminUser) []AdminUserResponse {
	result := make([]AdminUserResponse, len(users))
	for i, u := range users {
		result[i] = ToAdminUserResponse(u)
	}
	return result
}
