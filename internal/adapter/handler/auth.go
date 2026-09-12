package handler

import (
	"net/http"
	"time"

	"github.com/firereach/api/internal/adapter/dto"
	"github.com/firereach/api/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	pool      *pgxpool.Pool
	jwtSecret string
}

func NewAuthHandler(pool *pgxpool.Pool, jwtSecret string) *AuthHandler {
	return &AuthHandler{pool: pool, jwtSecret: jwtSecret}
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

	var id, passwordHash string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT id, password_hash FROM admin_users WHERE email = $1`, req.Email,
	).Scan(&id, &passwordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": id,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: tokenString})
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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	var id string
	err = h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		req.Email, string(hash),
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	c.JSON(http.StatusCreated, dto.CreatedAdminResponse{ID: id, Email: req.Email})
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
	var users []domain.AdminUser

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, email, created_at FROM admin_users`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var user domain.AdminUser
		err := rows.Scan(&user.ID, &user.Email, &user.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, dto.ToAdminUserListResponse(users))
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
	var count int
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM admin_users`,
	).Scan(&count)
	if err != nil {
		log.Error().Err(err).Msg("failed to check admin count")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check admin count"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "setup already completed — admin exists"})
		return
	}

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	var id string
	err = h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		req.Email, string(hash),
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	c.JSON(http.StatusCreated, dto.CreatedAdminResponse{ID: id, Email: req.Email})
}
