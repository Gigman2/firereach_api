package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/ai"
)

type AIHandler struct {
	askAI *ai.AskAI
}

func NewAIHandler(a *ai.AskAI) *AIHandler {
	return &AIHandler{askAI: a}
}

// Ask godoc
// @Summary      Ask the safety assistant a question
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        X-Device-Hash  header    string  false  "Opaque per-device identifier. Rate limiting keys on this header; without it the limit is shared across everyone behind the same IP."
// @Param        request  body      dto.AskAIRequest  true  "Question and optional topic"
// @Success      200      {object}  dto.AskAIResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      429      {object}  dto.ErrorResponse  "Rate limited"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /ai/ask [post]
func (h *AIHandler) Ask(c *gin.Context) {
	var req dto.AskAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answer, err := h.askAI.Execute(c.Request.Context(), req.Question, req.Topic)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrRateLimited) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.AskAIResponse{Answer: answer})
}
