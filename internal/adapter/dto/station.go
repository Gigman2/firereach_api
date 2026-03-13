package dto

import "github.com/firereach/api/internal/domain"

type ContactInput struct {
	Phone        string  `json:"phone" binding:"required"`
	ResponseRate float64 `json:"response_rate"`
	Active       bool    `json:"active"`
}

type CreateStationRequest struct {
	Name     string         `json:"name" binding:"required"`
	Region   string         `json:"region" binding:"required"`
	District string         `json:"district" binding:"required"`
	Lat      float64        `json:"lat" binding:"required"`
	Lng      float64        `json:"lng" binding:"required"`
	Contacts []ContactInput `json:"contacts" binding:"required,min=1"`
}

type UpdateStationRequest struct {
	Name     string         `json:"name"`
	Region   string         `json:"region"`
	District string         `json:"district"`
	Lat      float64        `json:"lat"`
	Lng      float64        `json:"lng"`
	Contacts []ContactInput `json:"contacts"`
}

func ContactInputsToDomain(inputs []ContactInput) []domain.StationContact {
	contacts := make([]domain.StationContact, len(inputs))
	for i, c := range inputs {
		contacts[i] = domain.StationContact{
			Phone:        c.Phone,
			ResponseRate: c.ResponseRate,
			Active:       bool(c.Active) ,
		}
	}
	return contacts
}

type StationContactResponse struct {
	Phone        string  `json:"phone"`
	ResponseRate float64 `json:"response_rate"`
	Active       bool    `json:"active"`
}

type StationResponse struct {
	ID       string                   `json:"id"`
	Name     string                   `json:"name"`
	Region   string                   `json:"region"`
	District string                   `json:"district"`
	Lat      float64                  `json:"lat"`
	Lng      float64                  `json:"lng"`
	Contacts []StationContactResponse `json:"contacts"`
	Distance float64                  `json:"distance,omitempty"`
}

func ToStationResponse(s domain.Station) StationResponse {
	contacts := make([]StationContactResponse, len(s.Contacts))
	for i, c := range s.Contacts {
		contacts[i] = StationContactResponse{
			Phone:        c.Phone,
			ResponseRate: c.ResponseRate,
			Active:       c.Active,
		}
	}
	return StationResponse{
		ID:       s.ID,
		Name:     s.Name,
		Region:   s.Region,
		District: s.District,
		Lat:      s.Lat,
		Lng:      s.Lng,
		Contacts: contacts,
	}
}

func ToStationListResponse(stations []domain.Station) []StationResponse {
	result := make([]StationResponse, len(stations))
	for i, s := range stations {
		result[i] = ToStationResponse(s)
	}
	return result
}
