package domain_test

import (
	"testing"

	"github.com/firereach/api/internal/domain"
)

func TestPrimaryPhone_HighestResponseRateWins(t *testing.T) {
	s := domain.Station{Contacts: []domain.StationContact{
		{Phone: "0302666576", ResponseRate: 0.5, Active: true},
		{Phone: "0299346018", ResponseRate: 1.0, Active: true},
		{Phone: "192", ResponseRate: 0.1, Active: true},
	}}
	if got := s.PrimaryPhone(); got != "0299346018" {
		t.Errorf("expected highest-rate contact, got %q", got)
	}
}

func TestPrimaryPhone_SkipsInactive(t *testing.T) {
	s := domain.Station{Contacts: []domain.StationContact{
		{Phone: "0302666576", ResponseRate: 1.0, Active: false},
		{Phone: "192", ResponseRate: 0.1, Active: true},
	}}
	if got := s.PrimaryPhone(); got != "192" {
		t.Errorf("expected inactive contact skipped, got %q", got)
	}
}

func TestPrimaryPhone_TiebreakIsDeterministic(t *testing.T) {
	a := domain.Station{Contacts: []domain.StationContact{
		{Phone: "0999999999", ResponseRate: 0.5, Active: true},
		{Phone: "0111111111", ResponseRate: 0.5, Active: true},
	}}
	b := domain.Station{Contacts: []domain.StationContact{
		{Phone: "0111111111", ResponseRate: 0.5, Active: true},
		{Phone: "0999999999", ResponseRate: 0.5, Active: true},
	}}
	if a.PrimaryPhone() != b.PrimaryPhone() {
		t.Fatalf("tiebreak not deterministic: %q vs %q", a.PrimaryPhone(), b.PrimaryPhone())
	}
	if got := a.PrimaryPhone(); got != "0111111111" {
		t.Errorf("expected lexicographically smallest phone on tie, got %q", got)
	}
}

func TestPrimaryPhone_NoContactsReturnsEmpty(t *testing.T) {
	if got := (domain.Station{}).PrimaryPhone(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestPrimaryPhone_AllInactiveReturnsEmpty(t *testing.T) {
	s := domain.Station{Contacts: []domain.StationContact{
		{Phone: "0302666576", ResponseRate: 1.0, Active: false},
	}}
	if got := s.PrimaryPhone(); got != "" {
		t.Errorf("expected empty string when all contacts inactive, got %q", got)
	}
}

func TestStation_HasDistanceMetersField(t *testing.T) {
	s := domain.Station{DistanceMeters: 2400}
	if s.DistanceMeters != 2400 {
		t.Errorf("expected 2400, got %d", s.DistanceMeters)
	}
}
