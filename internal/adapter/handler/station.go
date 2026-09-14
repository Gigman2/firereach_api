package handler

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/station"
	"github.com/rs/zerolog/log"
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

// ListNearest godoc
// @Summary      List nearest fire stations
// @Description  Returns active stations ordered by distance from the supplied coordinates, nearest first.
// @Tags         stations
// @Produce      json
// @Param        lat    query     number  true   "Latitude, -90 to 90"
// @Param        lng    query     number  true   "Longitude, -180 to 180"
// @Param        limit  query     integer false  "Maximum results (default 3)"
// @Success      200    {array}   dto.StationResponse
// @Failure      400    {object}  dto.ErrorResponse  "Missing, unparseable, NaN, or out-of-range coordinate"
// @Failure      500    {object}  dto.ErrorResponse
// @Router       /stations [get]
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

// GetByID godoc
// @Summary      Get a station by ID
// @Description  Returns one station. distance_meters is 0 here, since no reference coordinate is supplied.
// @Tags         stations
// @Produce      json
// @Param        id   path      string  true  "Station UUID"
// @Success      200  {object}  dto.StationResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /stations/{id} [get]
func (h *StationHandler) GetByID(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

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

// Create godoc
// @Summary      Create a fire station
// @Tags         admin, stations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateStationRequest  true  "Station details, at least one contact required"
// @Success      201      {object}  dto.StationResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/stations [post]
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
		Lat:      *req.Lat,
		Lng:      *req.Lng,
		Contacts: dto.ContactInputsToDomain(req.Contacts),
	}

	created, err := h.create.Execute(c.Request.Context(), s)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Error().Err(err).Msg("create station")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, dto.ToStationResponse(*created))
}

// Update godoc
// @Summary      Update a fire station
// @Tags         admin, stations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                    true  "Station UUID"
// @Param        request  body      dto.UpdateStationRequest  true  "Fields to update"
// @Success      200      {object}  dto.MessageResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      404      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/stations/{id} [patch]
func (h *StationHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.UpdateStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patch := station.StationPatch{
		Name:     req.Name,
		Region:   req.Region,
		District: req.District,
		Lat:      req.Lat,
		Lng:      req.Lng,
	}
	if req.Contacts != nil {
		patch.Contacts = dto.ContactInputsToDomain(req.Contacts)
	}

	if err := h.update.Execute(c.Request.Context(), id, patch); err != nil {
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

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "station updated"})
}

// Deactivate godoc
// @Summary      Deactivate a fire station
// @Description  Soft delete. The station stops appearing in nearest-station results.
// @Tags         admin, stations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Station UUID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /admin/stations/{id} [delete]
func (h *StationHandler) Deactivate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	if err := h.deactivate.Execute(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "station not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "station deactivated"})
}
