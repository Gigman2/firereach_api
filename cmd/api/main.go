package main

import (
	"log"

	"github.com/firereach/api/internal/adapter/handler"
	"github.com/firereach/api/internal/infra/claude"
	"github.com/firereach/api/internal/infra/config"
	"github.com/firereach/api/internal/infra/postgres"
	"github.com/firereach/api/internal/infra/repo"
	"github.com/firereach/api/internal/infra/router"

	aiuc "github.com/firereach/api/internal/usecase/ai"
	contentuc "github.com/firereach/api/internal/usecase/content"
	stationuc "github.com/firereach/api/internal/usecase/station"
	submissionuc "github.com/firereach/api/internal/usecase/submission"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Infrastructure — DB and external clients
	pool, err := postgres.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	claudeGW := claude.NewGateway(cfg.ClaudeAPIKey)

	// 3. Repositories — implement domain interfaces
	stationRepo := repo.NewStationRepo(pool)
	submissionRepo := repo.NewSubmissionRepo(pool)
	contentRepo := repo.NewContentRepo(pool)

	// 4. Use cases — injected with repositories
	listNearest := stationuc.NewListNearestStations(stationRepo)
	getStation := stationuc.NewGetStation(stationRepo)
	createStation := stationuc.NewCreateStation(stationRepo)
	updateStation := stationuc.NewUpdateStation(stationRepo)
	deactivateStation := stationuc.NewDeactivateStation(stationRepo)
	createSub := submissionuc.NewCreateSubmission(submissionRepo)
	listPending := submissionuc.NewListPending(submissionRepo)
	reviewSub := submissionuc.NewReviewSubmission(submissionRepo, stationRepo)
	listContent := contentuc.NewListContent(contentRepo)
	getContent := contentuc.NewGetContent(contentRepo)
	askAI := aiuc.NewAskAI(claudeGW)

	// 5. Handlers — injected with use cases
	stationH := handler.NewStationHandler(listNearest, getStation, createStation, updateStation, deactivateStation)
	submissionH := handler.NewSubmissionHandler(createSub, listPending, reviewSub)
	contentH := handler.NewContentHandler(listContent, getContent)
	aiH := handler.NewAIHandler(askAI)
	authH := handler.NewAuthHandler(pool, cfg.JWTSecret)

	// 6. Router — wire handlers, start server
	r := router.New(cfg, stationH, submissionH, contentH, aiH, authH)

	port := cfg.Port
	if port == "" {
		port = "9000"
	}

	log.Printf("starting server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
