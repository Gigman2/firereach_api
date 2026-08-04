package dto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
)

func sampleStation() domain.Station {
	return domain.Station{
		ID:             "11111111-1111-1111-1111-111111111111",
		Name:           "Accra City Fire Station",
		Region:         "Greater Accra Region",
		District:       "Accra Metropolitan District",
		Lat:            5.54931,
		Lng:            -0.20730,
		DistanceMeters: 2400,
		Contacts: []domain.StationContact{
			{Phone: "0302666576", ResponseRate: 1.0, Active: true},
			{Phone: "192", ResponseRate: 0.1, Active: true},
		},
	}
}

func TestToStationResponse_CarriesDistanceAndPrimaryPhone(t *testing.T) {
	got := dto.ToStationResponse(sampleStation())
	if got.DistanceMeters != 2400 {
		t.Errorf("expected 2400, got %d", got.DistanceMeters)
	}
	if got.PrimaryPhone != "0302666576" {
		t.Errorf("expected highest-rate contact, got %q", got.PrimaryPhone)
	}
	if len(got.Contacts) != 2 {
		t.Errorf("expected 2 contacts preserved, got %d", len(got.Contacts))
	}
}

func TestToStationResponse_ZeroDistanceIsStillSerialized(t *testing.T) {
	s := sampleStation()
	s.DistanceMeters = 0

	blob, err := json.Marshal(dto.ToStationResponse(s))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(blob), `"distance_meters":0`) {
		t.Errorf("zero distance must not be omitted, got %s", blob)
	}
}

func TestToStationResponse_PrimaryPhoneJSONTag(t *testing.T) {
	blob, err := json.Marshal(dto.ToStationResponse(sampleStation()))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(blob), `"primary_phone":"0302666576"`) {
		t.Errorf("expected primary_phone JSON tag with highest-rate contact, got %s", blob)
	}
}

func TestToStationResponse_EmptyPrimaryPhoneWhenNoActiveContacts(t *testing.T) {
	s := sampleStation()
	s.Contacts = []domain.StationContact{{Phone: "0302666576", ResponseRate: 1.0, Active: false}}

	if got := dto.ToStationResponse(s).PrimaryPhone; got != "" {
		t.Errorf("expected empty primary phone, got %q", got)
	}
}

func TestToStationListResponse_MapsEveryStation(t *testing.T) {
	got := dto.ToStationListResponse([]domain.Station{sampleStation(), sampleStation()})
	if len(got) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(got))
	}
	if got[1].PrimaryPhone != "0302666576" {
		t.Errorf("second entry not mapped, got %q", got[1].PrimaryPhone)
	}
}
