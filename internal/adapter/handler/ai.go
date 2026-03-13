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
