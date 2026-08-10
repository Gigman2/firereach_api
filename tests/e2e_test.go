package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/firereach/api/internal/adapter/handler"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/config"
	"github.com/firereach/api/internal/infra/router"
	"github.com/firereach/api/internal/usecase/ai"
	contentuc "github.com/firereach/api/internal/usecase/content"
	"github.com/firereach/api/internal/usecase/mocks"
	stationuc "github.com/firereach/api/internal/usecase/station"
	submissionuc "github.com/firereach/api/internal/usecase/submission"
	"github.com/gin-gonic/gin"
)

// testApp holds the test server and helper state.
type testApp struct {
	router      *gin.Engine
	stationRepo *mocks.StationRepo
	subRepo     *mocks.SubmissionRepo
	contentRepo *mocks.ContentRepo
	aiGateway   *mocks.AIGateway
}

func newTestApp() *testApp {
	return newTestAppWithEnv("")
}

func newTestAppWithEnv(env string) *testApp {
	gin.SetMode(gin.TestMode)

	stationRepo := &mocks.StationRepo{
		ListActiveFunc: func(ctx context.Context) ([]domain.Station, error) {
			return []domain.Station{
				{
					ID: "station-1", Name: "Accra Central", Region: "Greater Accra",
					District: "Accra Metropolitan", Lat: 5.55, Lng: -0.20, Active: true,
					Contacts:  []domain.StationContact{{Phone: "0302123456", ResponseRate: 0.95, Active: true}},
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				},
				{
					ID: "station-2", Name: "Kumasi Central", Region: "Ashanti",
					District: "Kumasi Metropolitan", Lat: 6.69, Lng: -1.62, Active: true,
					Contacts:  []domain.StationContact{{Phone: "0322345678", ResponseRate: 0.90, Active: true}},
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				},
				{
					ID: "station-3", Name: "Tema Industrial", Region: "Greater Accra",
					District: "Tema Metropolitan", Lat: 5.67, Lng: -0.01, Active: true,
					Contacts:  []domain.StationContact{{Phone: "0303456789", ResponseRate: 0.85, Active: true}},
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				},
			}, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*domain.Station, error) {
			if id == "station-1" {
				return &domain.Station{
					ID: "station-1", Name: "Accra Central", Region: "Greater Accra",
					District: "Accra Metropolitan", Lat: 5.55, Lng: -0.20, Active: true,
					Contacts:  []domain.StationContact{{Phone: "0302123456", ResponseRate: 0.95, Active: true}},
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}, nil
			}
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, s domain.Station) error {
			return nil
		},
		UpdateFunc: func(ctx context.Context, s domain.Station) error {
			return nil
		},
	}

	subRepo := &mocks.SubmissionRepo{
		CreateFunc: func(ctx context.Context, s domain.Submission) error {
			return nil
		},
		ListPendingFunc: func(ctx context.Context) ([]domain.Submission, error) {
			return []domain.Submission{
				{
					ID: "sub-1", Type: "phone_correction", SuggestedValue: "0302999999",
					DeviceHash: "abc123", Status: domain.SubmissionStatusPending,
					SubmittedAt: time.Now(),
				},
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
			if id == "sub-999" {
				return domain.ErrNotFound
			}
			return nil
		},
	}

	contentRepo := &mocks.ContentRepo{
		ListFunc: func(ctx context.Context, category, subcategory string) ([]domain.SafetyContent, error) {
			return []domain.SafetyContent{
				{ID: "content-1", Category: "hazard", Subcategory: "electrical", Title: "Electrical Fire Safety"},
				{ID: "content-2", Category: "first_aid", Subcategory: "burns", Title: "Burns Treatment"},
			}, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*domain.SafetyContent, error) {
			if id == "content-1" {
				return &domain.SafetyContent{
					ID: "content-1", Category: "hazard", Subcategory: "electrical",
					Title: "Electrical Fire Safety", Body: "Never use water on electrical fires.",
					Steps: []domain.Step{
						{Title: "Cut power", Body: "Isolate the circuit at the breaker."},
						{Title: "Use CO2 extinguisher", Body: "Aim at the base of the flames."},
						{Title: "Call emergency services", Body: "Dial the national emergency number."},
					},
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	}

	aiGateway := &mocks.AIGateway{
		AskFunc: func(ctx context.Context, question, topic string) (string, error) {
			return "In case of a fire, evacuate immediately and call emergency services.", nil
		},
	}

	// Wire use cases
	listNearest := stationuc.NewListNearestStations(stationRepo)
	getStation := stationuc.NewGetStation(stationRepo)
	createStation := stationuc.NewCreateStation(stationRepo)
	updateStation := stationuc.NewUpdateStation(stationRepo)
	deactivateStation := stationuc.NewDeactivateStation(stationRepo)
	createSub := submissionuc.NewCreateSubmission(subRepo)
	listPending := submissionuc.NewListPending(subRepo)
	reviewSub := submissionuc.NewReviewSubmission(subRepo, stationRepo)
	listContent := contentuc.NewListContent(contentRepo)
	getContent := contentuc.NewGetContent(contentRepo)
	askAI := ai.NewAskAI(aiGateway)

	// Wire handlers — AuthHandler needs a real pool, so we skip auth-dependent tests
	// that need DB and test the rest of the API
	stationH := handler.NewStationHandler(listNearest, getStation, createStation, updateStation, deactivateStation)
	submissionH := handler.NewSubmissionHandler(createSub, listPending, reviewSub)
	contentH := handler.NewContentHandler(listContent, getContent)
	aiH := handler.NewAIHandler(askAI)
	authH := handler.NewAuthHandler(nil, "test-jwt-secret")

	cfg := &config.Config{JWTSecret: "test-jwt-secret", Environment: env}
	r := router.New(cfg, stationH, submissionH, contentH, aiH, authH)

	return &testApp{
		router:      r,
		stationRepo: stationRepo,
		subRepo:     subRepo,
		contentRepo: contentRepo,
		aiGateway:   aiGateway,
	}
}

func (a *testApp) request(method, path string, body interface{}, headers ...map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	return w
}

func (a *testApp) adminToken() string {
	// Generate a valid JWT for testing protected routes
	token := generateTestJWT("test-admin-id", "test-jwt-secret")
	return "Bearer " + token
}

// ─────────────────────────────────────────────
// Station Endpoints
// ─────────────────────────────────────────────

func TestListNearestStations_OK(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/stations?lat=5.56&lng=-0.19", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var stations []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &stations)
	if len(stations) == 0 {
		t.Fatal("expected stations in response")
	}
	// Nearest to Accra (5.56, -0.19) should be Accra Central first
	if stations[0]["name"] != "Accra Central" {
		t.Errorf("expected Accra Central first, got %s", stations[0]["name"])
	}
}

func TestListNearestStations_WithLimit(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/stations?lat=5.56&lng=-0.19&limit=1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stations []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &stations)
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}
}

func TestListNearestStations_MissingLat(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/stations?lng=-0.19", nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListNearestStations_InvalidLng(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/stations?lat=5.56&lng=abc", nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// strconv.ParseFloat accepts "NaN", "Inf" and their variants, and every
// comparison against NaN is false — so a bare range check lets NaN through.
// A NaN coordinate makes every haversine result NaN, which rounds to a
// distance of 0 metres, so the sort order becomes arbitrary and the caller is
// handed a random station as their "nearest". Reject non-finite coordinates.
func TestListNearestStations_RejectsNonFiniteAndOutOfRange(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"lat NaN", "lat=NaN&lng=-0.19"},
		{"lat nan lowercase", "lat=nan&lng=-0.19"},
		{"lng NaN", "lat=5.56&lng=NaN"},
		{"lat Inf", "lat=Inf&lng=-0.19"},
		{"lng -Inf", "lat=5.56&lng=-Inf"},
		{"lat above range", "lat=999&lng=-0.19"},
		{"lat below range", "lat=-91&lng=-0.19"},
		{"lng above range", "lat=5.56&lng=181"},
		{"lng below range", "lat=5.56&lng=-181"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApp()
			w := app.request("GET", "/v1/stations?"+tc.query, nil)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %q, got %d", tc.query, w.Code)
			}
		})
	}
}

func TestListNearestStations_AcceptsRangeBoundaries(t *testing.T) {
	cases := []string{
		"lat=90&lng=180",
		"lat=-90&lng=-180",
		"lat=0&lng=0",
	}

	for _, query := range cases {
		t.Run(query, func(t *testing.T) {
			app := newTestApp()
			w := app.request("GET", "/v1/stations?"+query, nil)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200 for %q, got %d", query, w.Code)
			}
		})
	}
}

func TestGetStation_OK(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/stations/station-1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var station map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &station)
	if station["name"] != "Accra Central" {
		t.Errorf("expected Accra Central, got %s", station["name"])
	}
	contacts, ok := station["contacts"].([]interface{})
	if !ok || len(contacts) == 0 {
		t.Error("expected contacts in station response")
	}
}

func TestGetStation_NotFound(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/stations/nonexistent", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────
// Submission Endpoints
// ─────────────────────────────────────────────

func TestCreateSubmission_OK(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"type":            "phone_correction",
		"suggested_value": "0302999999",
		"device_hash":     "device-abc",
	}
	headers := map[string]string{"X-Device-Hash": "sub-ok-test"}
	w := app.request("POST", "/v1/submissions", body, headers)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSubmission_MissingType(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"suggested_value": "0302999999",
		"device_hash":     "device-abc",
	}
	headers := map[string]string{"X-Device-Hash": "sub-missing-type-test"}
	w := app.request("POST", "/v1/submissions", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSubmission_MissingDeviceHash(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"type":            "phone_correction",
		"suggested_value": "0302999999",
	}
	headers := map[string]string{"X-Device-Hash": "sub-missing-hash-test"}
	w := app.request("POST", "/v1/submissions", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────
// Content Endpoints
// ─────────────────────────────────────────────

func TestListContent_OK(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/content", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 content items, got %d", len(items))
	}
}

func TestListContent_FilterByCategory(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/content?category=hazard", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetContent_OK(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/content/content-1", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var content map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &content)
	if content["title"] != "Electrical Fire Safety" {
		t.Errorf("expected 'Electrical Fire Safety', got %s", content["title"])
	}
	steps, ok := content["steps"].([]interface{})
	if !ok || len(steps) != 3 {
		t.Error("expected 3 steps in content response")
	}
}

func TestGetContent_NotFound(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/content/nonexistent", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────
// AI Endpoint
// ─────────────────────────────────────────────

func TestAskAI_OK(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"question": "What should I do if there is a fire?",
		"topic":    "hazard",
	}
	headers := map[string]string{"X-Device-Hash": "ai-ok-test"}
	w := app.request("POST", "/v1/ai/ask", body, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["answer"] == "" {
		t.Error("expected non-empty answer")
	}
}

func TestAskAI_EmptyQuestion(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"question": "",
	}
	headers := map[string]string{"X-Device-Hash": "ai-empty-q-test"}
	w := app.request("POST", "/v1/ai/ask", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAskAI_MissingQuestion(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{}
	headers := map[string]string{"X-Device-Hash": "ai-missing-q-test"}
	w := app.request("POST", "/v1/ai/ask", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────
// Admin Endpoints — Auth Required
// ─────────────────────────────────────────────

func TestAdminRoutes_Unauthorized(t *testing.T) {
	app := newTestApp()

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/v1/admin/submissions"},
		{"PATCH", "/v1/admin/submissions/sub-1"},
		{"POST", "/v1/admin/stations"},
		{"PATCH", "/v1/admin/stations/station-1"},
		{"DELETE", "/v1/admin/stations/station-1"},
		{"POST", "/v1/admin/users"},
		{"GET", "/v1/admin/users"},
	}

	for _, tt := range tests {
		w := app.request(tt.method, tt.path, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tt.method, tt.path, w.Code)
		}
	}
}

func TestAdminRoutes_InvalidToken(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": "Bearer invalid-token"}
	w := app.request("GET", "/v1/admin/submissions", nil, headers)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminListPending_OK(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	w := app.request("GET", "/v1/admin/submissions", nil, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var subs []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &subs)
	if len(subs) != 1 {
		t.Fatalf("expected 1 pending submission, got %d", len(subs))
	}
	if subs[0]["status"] != "pending" {
		t.Errorf("expected status 'pending', got %s", subs[0]["status"])
	}
}

func TestAdminReviewSubmission_Approve(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"status":     "approved",
		"admin_note": "verified",
	}
	w := app.request("PATCH", "/v1/admin/submissions/sub-1", body, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminReviewSubmission_Reject(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"status":     "rejected",
		"admin_note": "duplicate submission",
	}
	w := app.request("PATCH", "/v1/admin/submissions/sub-1", body, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminReviewSubmission_InvalidStatus(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"status": "pending",
	}
	w := app.request("PATCH", "/v1/admin/submissions/sub-1", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminReviewSubmission_NotFound(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"status": "approved",
	}
	w := app.request("PATCH", "/v1/admin/submissions/sub-999", body, headers)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────
// Admin Station CRUD
// ─────────────────────────────────────────────

func TestAdminCreateStation_OK(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name":     "Takoradi Station",
		"region":   "Western",
		"district": "Sekondi-Takoradi",
		"lat":      4.93,
		"lng":      -1.77,
		"contacts": []map[string]interface{}{
			{"phone": "0312345678", "response_rate": 0.9, "active": true},
		},
	}
	w := app.request("POST", "/v1/admin/stations", body, headers)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var station map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &station)
	if station["name"] != "Takoradi Station" {
		t.Errorf("expected 'Takoradi Station', got %s", station["name"])
	}
	if station["id"] == "" {
		t.Error("expected station to have an ID")
	}
}

func TestAdminCreateStation_MissingName(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"region":   "Western",
		"district": "Sekondi-Takoradi",
		"lat":      4.93,
		"lng":      -1.77,
		"contacts": []map[string]interface{}{
			{"phone": "0312345678"},
		},
	}
	w := app.request("POST", "/v1/admin/stations", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminCreateStation_NoContacts(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name":     "Empty Station",
		"region":   "Western",
		"district": "Sekondi-Takoradi",
		"lat":      4.93,
		"lng":      -1.77,
		"contacts": []map[string]interface{}{},
	}
	w := app.request("POST", "/v1/admin/stations", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminCreateStation_Unauthorized(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"name":     "Takoradi Station",
		"region":   "Western",
		"district": "Sekondi-Takoradi",
		"lat":      4.93,
		"lng":      -1.77,
		"contacts": []map[string]interface{}{
			{"phone": "0312345678"},
		},
	}
	w := app.request("POST", "/v1/admin/stations", body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminUpdateStation_OK(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name": "Accra Central (Updated)",
	}
	w := app.request("PATCH", "/v1/admin/stations/station-1", body, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminUpdateStation_NotFound(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name": "Ghost Station",
	}
	w := app.request("PATCH", "/v1/admin/stations/nonexistent", body, headers)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeactivateStation_OK(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	w := app.request("DELETE", "/v1/admin/stations/station-1", nil, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeactivateStation_NotFound(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	w := app.request("DELETE", "/v1/admin/stations/nonexistent", nil, headers)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────
// CORS
// ─────────────────────────────────────────────

func TestCORS_Preflight(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest("OPTIONS", "/v1/stations?lat=5.56&lng=-0.19", nil)
	req.Header.Set("Origin", "http://admin.firereach.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	w := httptest.NewRecorder()
	app.router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}

// ─────────────────────────────────────────────
// 404 for unknown routes
// ─────────────────────────────────────────────

func TestUnknownRoute_404(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/nonexistent", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
