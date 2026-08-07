package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/submission"
)

type SubmissionHandler struct {
	create      *submission.CreateSubmission
	listPending *submission.ListPending
	review      *submission.ReviewSubmission
}

func NewSubmissionHandler(c *submission.CreateSubmission, lp *submission.ListPending, r *submission.ReviewSubmission) *SubmissionHandler {
	return &SubmissionHandler{create: c, listPending: lp, review: r}
}

// Create godoc
// @Summary      Submit a correction to station data
// @Description  Community-reported correction. Enters a pending queue for admin review.
// @Tags         submissions
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateSubmissionRequest  true  "Correction details"
// @Success      201      {object}  dto.MessageResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      429      {object}  dto.ErrorResponse  "Rate limited"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /submissions [post]
func (h *SubmissionHandler) Create(c *gin.Context) {
	var req dto.CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s := domain.Submission{
		StationID:      req.StationID,
		Type:           req.Type,
		SuggestedValue: req.SuggestedValue,
		Note:           req.Note,
		DeviceHash:     req.DeviceHash,
	}

	if err := h.create.Execute(c.Request.Context(), s); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, dto.MessageResponse{Message: "submission created"})
}

func (h *SubmissionHandler) ListPending(c *gin.Context) {
	subs, err := h.listPending.Execute(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.ToSubmissionListResponse(subs))
}

func (h *SubmissionHandler) Review(c *gin.Context) {
	id := c.Param("id")

	var req dto.ReviewSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := domain.SubmissionStatus(req.Status)
	if err := h.review.Execute(c.Request.Context(), id, status, req.AdminNote); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "submission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "submission reviewed"})
}
