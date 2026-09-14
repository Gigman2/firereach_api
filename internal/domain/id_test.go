package domain_test

import (
	"testing"

	"github.com/firereach/api/internal/domain"
)

func TestValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"5a0c5e1e-0000-4000-8000-000000000001", true},
		{"5A0C5E1E-0000-4000-8000-00000000000F", true},
		{"", false},
		{"not-a-uuid", false},
		{"station-1", false},
		// Postgres accepts these spellings, but the API never issues them.
		{"5a0c5e1e000040008000000000000001", false},
		{"{5a0c5e1e-0000-4000-8000-000000000001}", false},
		// Postgres rejects these, which is the 500 the check prevents.
		{"urn:uuid:5a0c5e1e-0000-4000-8000-000000000001", false},
		{"5a0c5e1e-0000-4000-8000-00000000000g", false},
		{"5a0c5e1e-0000-4000-8000-0000000000001", false},
		{"5a0c5e1e_0000_4000_8000_000000000001", false},
	}
	for _, tt := range tests {
		if got := domain.ValidID(tt.id); got != tt.want {
			t.Errorf("ValidID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
