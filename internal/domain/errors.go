package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrRateLimited   = errors.New("rate limited")
	ErrUnauthorized  = errors.New("unauthorized")
)
