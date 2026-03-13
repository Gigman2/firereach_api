package router

import (
	"github.com/gin-gonic/gin"

	"github.com/firereach/api/internal/adapter/handler"
	"github.com/firereach/api/internal/infra/config"
	"github.com/firereach/api/internal/infra/router/middleware"
)

func New(
	cfg *config.Config,
	stationH *handler.StationHandler,
	submissionH *handler.SubmissionHandler,
	contentH *handler.ContentHandler,
	aiH *handler.AIHandler,
	authH *handler.AuthHandler,
) *gin.Engine {
	r := gin.Default()
	r.RedirectTrailingSlash = false

	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	v1 := r.Group("/v1")
	{
		stations := v1.Group("/stations")
		stations.GET("", stationH.ListNearest)
		stations.GET("/:id", stationH.GetByID)

		v1.POST("/submissions", middleware.RateLimit(), submissionH.Create)

		v1.POST("/ai/ask", middleware.RateLimit(), aiH.Ask)

		content := v1.Group("/content")
		content.GET("", contentH.List)
		content.GET("/:id", contentH.GetByID)

		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/setup", authH.Setup)

		// Admin routes
		admin := v1.Group("/admin", middleware.Auth(cfg.JWTSecret))

		adminSubmissions := admin.Group("/submissions")
		adminSubmissions.GET("", submissionH.ListPending)
		adminSubmissions.PATCH("/:id", submissionH.Review)
		
		adminUsers := admin.Group("/users")
		adminUsers.POST("", authH.Register)
		adminUsers.GET("", authH.List)

		adminStations := admin.Group("/stations")
		adminStations.POST("", stationH.Create)
		adminStations.PATCH("/:id", stationH.Update)
		adminStations.DELETE("/:id", stationH.Deactivate)
	}

	return r
}
