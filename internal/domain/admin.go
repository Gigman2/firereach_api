package domain

import "time"

type AdminUser struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
