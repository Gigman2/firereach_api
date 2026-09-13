package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/admin"
)

// setupCompletedMsg is the 403 answer setup has always given once any admin
// exists.
const setupCompletedMsg = "setup already completed — admin exists"

type AuthHandler struct {
	login  *admin.Login
	create *admin.CreateAdmin
	list   *admin.ListAdmins
	setup  *admin.Setup
}

func NewAuthHandler(login *admin.Login, create *admin.CreateAdmin, list *admin.ListAdmins, setup *admin.Setup) *AuthHandler {
	return &AuthHandler{login: login, create: create, list: list, setup: setup}
}

// Login godoc
// @Summary      Authenticate an admin
// @Description  Returns a JWT valid for 24 hours.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Credentials"
// @Success      200      {object}  dto.LoginResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse  "Invalid credentials"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.login.Execute(c.Request.Context(), req.Email, req.Password)
	if errors.Is(err, domain.ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token})
}

// Register godoc
// @Summary      Create an admin account
// @Tags         admin, auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.RegisterRequest  true  "Credentials"
// @Success      201      {object}  dto.CreatedAdminResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      409      {object}  dto.ErrorResponse  "Email already registered"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/users [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.create.Execute(c.Request.Context(), req.Email, req.Password)
	if createFailed(c, err, "register admin") {
		return
	}

	c.JSON(http.StatusCreated, dto.CreatedAdminResponse{ID: created.ID, Email: created.Email})
}

// List godoc
// @Summary      List admin accounts
// @Tags         admin, auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   dto.AdminUserResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /admin/users [get]
func (h *AuthHandler) List(c *gin.Context) {
	admins, err := h.list.Execute(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("list admins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, dto.ToAdminUserListResponse(admins))
}

// Setup godoc
// @Summary      Create the first admin account
// @Description  One-time bootstrap. Returns 403 once any admin exists.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest  true  "Credentials for the first admin"
// @Success      201      {object}  dto.CreatedAdminResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      403      {object}  dto.ErrorResponse  "Setup already completed"
// @Failure      409      {object}  dto.ErrorResponse  "Email already registered"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /auth/setup [post]
func (h *AuthHandler) Setup(c *gin.Context) {
	// Answer 403 before reading the body, as setup always has. This early
	// check is only the fast path: two requests can both pass it, and the
	// atomic create below is what guarantees a single first admin.
	done, err := h.setup.Completed(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to check admin count")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check admin count"})
		return
	}
	if done {
		c.JSON(http.StatusForbidden, gin.H{"error": setupCompletedMsg})
		return
	}

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.setup.Execute(c.Request.Context(), req.Email, req.Password)
	if createFailed(c, err, "setup first admin") {
		return
	}

	c.JSON(http.StatusCreated, dto.CreatedAdminResponse{ID: created.ID, Email: created.Email})
}

// createFailed answers a failed admin creation, shared by register and setup,
// and reports whether it answered. A duplicate email and a lost setup race
// keep their own answers; anything else is a server fault, logged, and never
// dressed up as a duplicate.
func createFailed(c *gin.Context, err error, op string) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, admin.ErrHashPassword):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
	case errors.Is(err, domain.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
	case errors.Is(err, domain.ErrSetupCompleted):
		c.JSON(http.StatusForbidden, gin.H{"error": setupCompletedMsg})
	default:
		log.Error().Err(err).Msg(op)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
	return true
}
