package handler

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/station"
)

type StationHandler struct {
	listNearest *station.ListNearestStations
	getByID     *station.GetStation
	create      *station.CreateStation
	update      *station.UpdateStation
	deactivate  *station.DeactivateStation
}

func NewStationHandler(
	ln *station.ListNearestStations,
	g *station.GetStation,
	cr *station.CreateStation,
	up *station.UpdateStation,
	de *station.DeactivateStation,
) *StationHandler {
	return &StationHandler{
		listNearest: ln,
		getByID:     g,
		create:      cr,
		update:      up,
		deactivate:  de,
	}
}

func (h *StationHandler) ListNearest(c *gin.Context) {
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat parameter"})
		return
	}
	// NaN needs its own check: ParseFloat accepts "NaN", and every comparison
	// against NaN is false, so a bare range test lets it through. It would then
	// poison every haversine result, rounding each distance to 0 and handing the
	// caller an arbitrary station as their "nearest". (±Inf fails the range test
	// on its own.)
	if math.IsNaN(lat) || lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat parameter"})
		return
	}
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lng parameter"})
		return
	}
	if math.IsNaN(lng) || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lng parameter"})
		return
	}

	limit := 3
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	stations, err := h.listNearest.Execute(c.Request.Context(), lat, lng, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.ToStationListResponse(stations))
}

func (h *StationHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	s, err := h.getByID.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "station not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.ToStationResponse(*s))
}

func (h *StationHandler) Create(c *gin.Context) {
	var req dto.CreateStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s := domain.Station{
		Name:     req.Name,
		Region:   req.Region,
		District: req.District,
		Lat:      req.Lat,
		Lng:      req.Lng,
		Contacts: dto.ContactInputsToDomain(req.Contacts),
	}

	created, err := h.create.Execute(c.Request.Context(), s)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fmt.Printf("create station error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, dto.ToStationResponse(*created))
}

func (h *StationHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s := domain.Station{
		ID:       id,
		Name:     req.Name,
		Region:   req.Region,
		District: req.District,
		Lat:      req.Lat,
		Lng:      req.Lng,
	}
	if req.Contacts != nil {
		s.Contacts = dto.ContactInputsToDomain(req.Contacts)
	}

	if err := h.update.Execute(c.Request.Context(), s); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "station not found"})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "station updated"})
}

func (h *StationHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")

	if err := h.deactivate.Execute(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "station not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "station deactivated"})
}
