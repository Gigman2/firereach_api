package domain_test

import (
	"errors"
	"testing"

	"github.com/firereach/api/internal/domain"
)

func TestDomainErrorsAreDistinct(t *testing.T) {
	errs := []error{
		domain.ErrNotFound,
		domain.ErrAlreadyExists,
		domain.ErrInvalidInput,
		domain.ErrRateLimited,
		domain.ErrUnauthorized,
	}

	for i, a := range errs {
		for j, b := range errs {
			if i != j && errors.Is(a, b) {
				t.Errorf("expected %v and %v to be distinct", a, b)
			}
		}
	}
}
