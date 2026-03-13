package dto

import "github.com/firereach/api/internal/domain"

type CreateSubmissionRequest struct {
	StationID      *string `json:"station_id"`
	Type           string  `json:"type" binding:"required"`
	SuggestedValue string  `json:"suggested_value" binding:"required"`
	Note           string  `json:"note"`
	DeviceHash     string  `json:"device_hash" binding:"required"`
}

type ReviewSubmissionRequest struct {
	Status    string `json:"status" binding:"required"`
	AdminNote string `json:"admin_note"`
}

type SubmissionResponse struct {
	ID             string  `json:"id"`
	StationID      *string `json:"station_id,omitempty"`
	Type           string  `json:"type"`
	SuggestedValue string  `json:"suggested_value"`
	Note           string  `json:"note,omitempty"`
	DeviceHash     string  `json:"device_hash"`
	Status         string  `json:"status"`
	SubmittedAt    string  `json:"submitted_at"`
	AdminNote      string  `json:"admin_note,omitempty"`
}

func ToSubmissionResponse(s domain.Submission) SubmissionResponse {
	return SubmissionResponse{
		ID:             s.ID,
		StationID:      s.StationID,
		Type:           s.Type,
		SuggestedValue: s.SuggestedValue,
		Note:           s.Note,
		DeviceHash:     s.DeviceHash,
		Status:         string(s.Status),
		SubmittedAt:    s.SubmittedAt.Format("2006-01-02T15:04:05Z"),
		AdminNote:      s.AdminNote,
	}
}

func ToSubmissionListResponse(subs []domain.Submission) []SubmissionResponse {
	result := make([]SubmissionResponse, len(subs))
	for i, s := range subs {
		result[i] = ToSubmissionResponse(s)
	}
	return result
}
