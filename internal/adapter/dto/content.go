package dto

import "github.com/firereach/api/internal/domain"

type ContentResponse struct {
	ID               string   `json:"id"`
	Category         string   `json:"category"`
	Subcategory      string   `json:"subcategory"`
	Title            string   `json:"title"`
	Body             string   `json:"body"`
	Steps            []string `json:"steps,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	ContextualTrigger string  `json:"contextual_trigger,omitempty"`
}

type AskAIRequest struct {
	Question string `json:"question" binding:"required"`
	Topic    string `json:"topic"`
}

type AskAIResponse struct {
	Answer string `json:"answer"`
}

func ToContentResponse(c domain.SafetyContent) ContentResponse {
	return ContentResponse{
		ID:               c.ID,
		Category:         c.Category,
		Subcategory:      c.Subcategory,
		Title:            c.Title,
		Body:             c.Body,
		Steps:            c.Steps,
		Tags:             c.Tags,
		ContextualTrigger: c.ContextualTrigger,
	}
}

func ToContentListResponse(items []domain.SafetyContent) []ContentResponse {
	result := make([]ContentResponse, len(items))
	for i, c := range items {
		result[i] = ToContentResponse(c)
	}
	return result
}
