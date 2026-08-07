package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/content"
)

type ContentHandler struct {
	list    *content.ListContent
	getByID *content.GetContent
}

func NewContentHandler(l *content.ListContent, g *content.GetContent) *ContentHandler {
	return &ContentHandler{list: l, getByID: g}
}

// List godoc
// @Summary      List safety content
// @Tags         content
// @Produce      json
// @Param        category     query     string  false  "Filter by category"
// @Param        subcategory  query     string  false  "Filter by subcategory"
// @Success      200          {array}   dto.ContentResponse
// @Failure      500          {object}  dto.ErrorResponse
// @Router       /content [get]
func (h *ContentHandler) List(c *gin.Context) {
	category := c.Query("category")
	subcategory := c.Query("subcategory")

	items, err := h.list.Execute(c.Request.Context(), category, subcategory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.ToContentListResponse(items))
}

// GetByID godoc
// @Summary      Get a safety content item by ID
// @Tags         content
// @Produce      json
// @Param        id   path      string  true  "Content ID"
// @Success      200  {object}  dto.ContentResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /content/{id} [get]
func (h *ContentHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	item, err := h.getByID.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.ToContentResponse(*item))
}
