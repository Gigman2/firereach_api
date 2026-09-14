package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/firereach/api/internal/adapter/handler"
	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/infra/config"
	"github.com/firereach/api/internal/infra/router"
	adminuc "github.com/firereach/api/internal/usecase/admin"
	"github.com/firereach/api/internal/usecase/ai"
	contentuc "github.com/firereach/api/internal/usecase/content"
	"github.com/firereach/api/internal/usecase/mocks"
	stationuc "github.com/firereach/api/internal/usecase/station"
	submissionuc "github.com/firereach/api/internal/usecase/submission"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// testApp holds the test server and helper state.
type testApp struct {
	router      *gin.Engine
	stationRepo *mocks.StationRepo
	subRepo     *mocks.SubmissionRepo
	contentRepo *mocks.ContentRepo
	aiGateway   *mocks.AIGateway
	adminRepo   *mocks.AdminRepo
	adminStore  *adminStore
}

// Ids for the mock repositories. Handlers refuse anything that is not a UUID
// before a repository sees it, so fixtures use well-formed ones; unknownID is
// well-formed and matches nothing.
const (
	testStationID    = "5a0c5e1e-0000-4000-8000-000000000001"
	testContentID    = "5a0c5e1e-0000-4000-8000-000000000002"
	testSubmissionID = "5a0c5e1e-0000-4000-8000-000000000003"
	unknownID        = "5a0c5e1e-0000-4000-8000-0000000000ff"
)

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
			if id == testStationID {
				return &domain.Station{
					ID: testStationID, Name: "Accra Central", Region: "Greater Accra",
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
					ID: testSubmissionID, Type: "wrong_phone", SuggestedValue: "0302999999",
					DeviceHash: "abc123", Status: domain.SubmissionStatusPending,
					SubmittedAt: time.Now(),
				},
			}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.SubmissionStatus, adminNote string) error {
			if id == unknownID {
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
			if id == testContentID {
				return &domain.SafetyContent{
					ID: testContentID, Category: "hazard", Subcategory: "electrical",
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
		AskFunc: func(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
			return domain.AIResponse{
				Kind:  domain.KindEmergency,
				Title: "ACTIVE EMERGENCY",
				Body:  "In case of a fire, evacuate immediately and call emergency services.",
				Items: []domain.AIStep{{Body: "Get out of the building"}},
			}, nil
		},
	}

	// An in-memory admin table behind the mock repository, seeded with one
	// admin, so login, register, list and setup run through the real
	// usecases and handler. Tests swap single functions to simulate a
	// database failure or a lost setup race, or empty the store to act as a
	// fresh system.
	store := newAdminStore()
	adminRepo := store.repo()

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
	loginAdmin := adminuc.NewLogin(adminRepo, "test-jwt-secret")
	createAdmin := adminuc.NewCreateAdmin(adminRepo)
	listAdmins := adminuc.NewListAdmins(adminRepo)
	setupAdmin := adminuc.NewSetup(adminRepo)

	// Wire handlers
	stationH := handler.NewStationHandler(listNearest, getStation, createStation, updateStation, deactivateStation)
	submissionH := handler.NewSubmissionHandler(createSub, listPending, reviewSub)
	contentH := handler.NewContentHandler(listContent, getContent)
	aiH := handler.NewAIHandler(askAI)
	authH := handler.NewAuthHandler(loginAdmin, createAdmin, listAdmins, setupAdmin)

	cfg := &config.Config{JWTSecret: "test-jwt-secret", Environment: env}
	r := router.New(cfg, stationH, submissionH, contentH, aiH, authH)

	return &testApp{
		router:      r,
		stationRepo: stationRepo,
		subRepo:     subRepo,
		contentRepo: contentRepo,
		aiGateway:   aiGateway,
		adminRepo:   adminRepo,
		adminStore:  store,
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
	w := app.request("GET", "/v1/stations/"+testStationID, nil)

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
	w := app.request("GET", "/v1/stations/"+unknownID, nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// A malformed id is refused before it reaches a repository. Postgres rejects a
// non-UUID with an error the repositories report as a server fault, so these
// used to answer 500 instead of saying what was wrong with the request.
func TestMalformedIDIsBadRequest(t *testing.T) {
	app := newTestApp()
	admin := map[string]string{"Authorization": app.adminToken()}

	tests := []struct {
		method  string
		path    string
		body    interface{}
		headers map[string]string
	}{
		{"GET", "/v1/stations/not-a-uuid", nil, nil},
		{"GET", "/v1/content/not-a-uuid", nil, nil},
		{"PATCH", "/v1/admin/submissions/not-a-uuid", map[string]interface{}{"status": "approved"}, admin},
		{"PATCH", "/v1/admin/stations/not-a-uuid", map[string]interface{}{"name": "Renamed"}, admin},
		{"DELETE", "/v1/admin/stations/not-a-uuid", nil, admin},
	}
	for _, tt := range tests {
		w := app.request(tt.method, tt.path, tt.body, tt.headers)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: expected 400, got %d: %s", tt.method, tt.path, w.Code, w.Body.String())
		}
	}
}

// ─────────────────────────────────────────────
// Submission Endpoints
// ─────────────────────────────────────────────

func TestCreateSubmission_OK(t *testing.T) {
	app := newTestApp()
	body := map[string]interface{}{
		"type":            "wrong_phone",
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
		"type":            "wrong_phone",
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
	w := app.request("GET", "/v1/content/"+testContentID, nil)

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
	w := app.request("GET", "/v1/content/"+unknownID, nil)

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
		t.Error("expected non-empty flattened answer")
	}
	if resp["kind"] != "emergency" {
		t.Errorf("expected kind=emergency, got %v", resp["kind"])
	}
	if resp["body"] == "" {
		t.Error("expected non-empty body")
	}
	items, ok := resp["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Errorf("expected one item, got %v", resp["items"])
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
		{"PATCH", "/v1/admin/submissions/" + testSubmissionID},
		{"POST", "/v1/admin/stations"},
		{"PATCH", "/v1/admin/stations/" + testStationID},
		{"DELETE", "/v1/admin/stations/" + testStationID},
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
	w := app.request("PATCH", "/v1/admin/submissions/"+testSubmissionID, body, headers)

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
	w := app.request("PATCH", "/v1/admin/submissions/"+testSubmissionID, body, headers)

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
	w := app.request("PATCH", "/v1/admin/submissions/"+testSubmissionID, body, headers)

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
	w := app.request("PATCH", "/v1/admin/submissions/"+unknownID, body, headers)

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

// A longitude of exactly 0 is Tema, on the prime meridian. Sending 0 for a
// float64 that gin marks `required` reads as "missing", so this used to be
// refused with a 400.
func TestAdminCreateStation_AcceptsZeroLongitude(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name":     "Tema Station",
		"region":   "Greater Accra",
		"district": "Tema Metropolitan",
		"lat":      5.67,
		"lng":      0,
		"contacts": []map[string]interface{}{
			{"phone": "0303456789", "response_rate": 0.9, "active": true},
		},
	}
	w := app.request("POST", "/v1/admin/stations", body, headers)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminCreateStation_RejectsCoordinatesOffTheEarth(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name":     "Nowhere Station",
		"region":   "Western",
		"district": "Sekondi-Takoradi",
		"lat":      91,
		"lng":      -1.77,
		"contacts": []map[string]interface{}{{"phone": "0312345678"}},
	}
	w := app.request("POST", "/v1/admin/stations", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Moving a station onto the prime meridian. The merge used to skip any zero, so
// the new longitude was dropped and the answer still said "station updated".
func TestAdminUpdateStation_AppliesAZeroCoordinate(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	var saved domain.Station
	app.stationRepo.UpdateFunc = func(ctx context.Context, s domain.Station) error {
		saved = s
		return nil
	}

	w := app.request("PATCH", "/v1/admin/stations/"+testStationID, map[string]interface{}{"lng": 0}, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if saved.Lng != 0 {
		t.Errorf("lng = %v, want 0", saved.Lng)
	}
	if saved.Lat != 5.55 {
		t.Errorf("lat = %v, want the existing 5.55", saved.Lat)
	}
}

func TestAdminUpdateStation_OK(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"name": "Accra Central (Updated)",
	}
	w := app.request("PATCH", "/v1/admin/stations/"+testStationID, body, headers)

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
	w := app.request("PATCH", "/v1/admin/stations/"+unknownID, body, headers)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeactivateStation_OK(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	w := app.request("DELETE", "/v1/admin/stations/"+testStationID, nil, headers)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminDeactivateStation_NotFound(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	w := app.request("DELETE", "/v1/admin/stations/"+unknownID, nil, headers)

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

// ─────────────────────────────────────────────
// Admin Auth
// ─────────────────────────────────────────────

const (
	testAdminEmail    = "admin@firereach.test"
	testAdminPassword = "correct-horse-battery"
)

// adminStore is the in-memory admin_users table behind the mock repository.
type adminStore struct {
	admins []domain.AdminUser
}

func newAdminStore() *adminStore {
	hash, err := bcrypt.GenerateFromPassword([]byte(testAdminPassword), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return &adminStore{admins: []domain.AdminUser{{
		ID: "admin-1", Email: testAdminEmail, PasswordHash: string(hash),
		CreatedAt: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC),
	}}}
}

func (s *adminStore) insert(email, hash string) *domain.AdminUser {
	a := domain.AdminUser{
		ID: fmt.Sprintf("admin-%d", len(s.admins)+1), Email: email, PasswordHash: hash,
		CreatedAt: time.Now().UTC(),
	}
	s.admins = append(s.admins, a)
	return &a
}

func (s *adminStore) repo() *mocks.AdminRepo {
	return &mocks.AdminRepo{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.AdminUser, error) {
			for _, a := range s.admins {
				if a.Email == email {
					found := a
					return &found, nil
				}
			}
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
			for _, a := range s.admins {
				if a.Email == email {
					return nil, domain.ErrAlreadyExists
				}
			}
			return s.insert(email, hash), nil
		},
		ListFunc: func(ctx context.Context) ([]domain.AdminUser, error) {
			return append([]domain.AdminUser(nil), s.admins...), nil
		},
		CountFunc: func(ctx context.Context) (int, error) {
			return len(s.admins), nil
		},
		CreateFirstFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
			if len(s.admins) > 0 {
				return nil, domain.ErrSetupCompleted
			}
			return s.insert(email, hash), nil
		},
	}
}

var errDatabaseDown = errors.New("connection refused")

func assertError(t *testing.T, w *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", w.Code, status, w.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not a JSON error: %s", w.Body)
	}
	if body["error"] != msg {
		t.Fatalf("error = %q, want %q", body["error"], msg)
	}
}

func login(app *testApp, email, password string) *httptest.ResponseRecorder {
	return app.request("POST", "/v1/auth/login", map[string]string{"email": email, "password": password})
}

func TestLogin_OK_TokenOpensTheAdminRoutes(t *testing.T) {
	app := newTestApp()
	w := login(app, testAdminEmail, testAdminPassword)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Token == "" {
		t.Fatalf("no token in %s", w.Body)
	}
	w = app.request("GET", "/v1/admin/users", nil, map[string]string{"Authorization": "Bearer " + resp.Token})
	if w.Code != http.StatusOK {
		t.Fatalf("the admin routes rejected the token login issued: %d %s", w.Code, w.Body)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	assertError(t, login(newTestApp(), testAdminEmail, "not-the-password"), http.StatusUnauthorized, "invalid credentials")
}

func TestLogin_UnknownEmail(t *testing.T) {
	assertError(t, login(newTestApp(), "nobody@firereach.test", testAdminPassword), http.StatusUnauthorized, "invalid credentials")
}

// The first bug this refactor fixes: any database error used to come back as
// "invalid credentials", so an outage looked like a wrong password.
func TestLogin_DatabaseDownIs500NotBadCredentials(t *testing.T) {
	app := newTestApp()
	app.adminRepo.GetByEmailFunc = func(ctx context.Context, email string) (*domain.AdminUser, error) {
		return nil, errDatabaseDown
	}
	assertError(t, login(app, testAdminEmail, testAdminPassword), http.StatusInternalServerError, "internal server error")
}

func TestLogin_MissingPassword(t *testing.T) {
	w := newTestApp().request("POST", "/v1/auth/login", map[string]string{"email": testAdminEmail})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestRegister_OK_ThenTheNewAdminCanLogIn(t *testing.T) {
	app := newTestApp()
	auth := map[string]string{"Authorization": app.adminToken()}
	w := app.request("POST", "/v1/admin/users", map[string]string{"email": "new@firereach.test", "password": "a-long-password"}, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	var created map[string]string
	json.Unmarshal(w.Body.Bytes(), &created)
	if created["id"] == "" || created["email"] != "new@firereach.test" {
		t.Fatalf("unexpected body %s", w.Body)
	}
	if w := login(app, "new@firereach.test", "a-long-password"); w.Code != http.StatusOK {
		t.Fatalf("the new admin cannot log in: %d %s", w.Code, w.Body)
	}
}

// bcrypt reads at most 72 bytes. A longer password is the caller's mistake, so
// it is answered 400; it used to surface as "failed to hash password" with a 500.
func TestRegisterAdmin_PasswordOver72BytesIsBadRequest(t *testing.T) {
	app := newTestApp()
	headers := map[string]string{"Authorization": app.adminToken()}
	body := map[string]interface{}{
		"email":    "new@firereach.test",
		"password": strings.Repeat("x", 73),
	}
	w := app.request("POST", "/v1/admin/users", body, headers)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	app := newTestApp()
	w := app.request("POST", "/v1/admin/users", map[string]string{"email": testAdminEmail, "password": "a-long-password"},
		map[string]string{"Authorization": app.adminToken()})
	assertError(t, w, http.StatusConflict, "email already registered")
}

// The second bug this refactor fixes: any insert failure used to come back as
// "email already registered", so an outage looked like a duplicate.
func TestRegister_DatabaseDownIs500NotConflict(t *testing.T) {
	app := newTestApp()
	app.adminRepo.CreateFunc = func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		return nil, errDatabaseDown
	}
	w := app.request("POST", "/v1/admin/users", map[string]string{"email": "new@firereach.test", "password": "a-long-password"},
		map[string]string{"Authorization": app.adminToken()})
	assertError(t, w, http.StatusInternalServerError, "internal server error")
}

func TestRegister_ShortPasswordRejected(t *testing.T) {
	app := newTestApp()
	w := app.request("POST", "/v1/admin/users", map[string]string{"email": "new@firereach.test", "password": "short"},
		map[string]string{"Authorization": app.adminToken()})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListAdmins_OK_NeverReturnsPasswordHashes(t *testing.T) {
	app := newTestApp()
	w := app.request("GET", "/v1/admin/users", nil, map[string]string{"Authorization": app.adminToken()})
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	var list []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("not a JSON array: %s", w.Body)
	}
	if len(list) != 1 || list[0]["id"] != "admin-1" || list[0]["email"] != testAdminEmail || list[0]["created_at"] != "2026-09-01T08:00:00Z" {
		t.Fatalf("unexpected list %s", w.Body)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("$2a$")) || bytes.Contains(w.Body.Bytes(), []byte("password")) {
		t.Fatalf("the list leaks a password or hash: %s", w.Body)
	}
}

func TestListAdmins_DatabaseDown(t *testing.T) {
	app := newTestApp()
	app.adminRepo.ListFunc = func(ctx context.Context) ([]domain.AdminUser, error) { return nil, errDatabaseDown }
	w := app.request("GET", "/v1/admin/users", nil, map[string]string{"Authorization": app.adminToken()})
	assertError(t, w, http.StatusInternalServerError, "failed to list users")
}

func TestSetup_OK_OnAFreshSystem(t *testing.T) {
	app := newTestApp()
	app.adminStore.admins = nil
	w := app.request("POST", "/v1/auth/setup", map[string]string{"email": "first@firereach.test", "password": "a-long-password"})
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: %d %s", w.Code, w.Body)
	}
	if w := login(app, "first@firereach.test", "a-long-password"); w.Code != http.StatusOK {
		t.Fatalf("the first admin cannot log in: %d %s", w.Code, w.Body)
	}
}

// Setup answers 403 as soon as any admin exists, before it even reads the
// body, exactly as it did before the refactor.
func TestSetup_OnceAnAdminExistsIs403EvenForABadBody(t *testing.T) {
	w := newTestApp().request("POST", "/v1/auth/setup", map[string]string{})
	assertError(t, w, http.StatusForbidden, "setup already completed — admin exists")
}

// The third bug this refactor fixes: setup counted, then inserted, so two
// requests at once could both succeed. Losing that race now answers 403.
func TestSetup_LosingTheRaceIs403(t *testing.T) {
	app := newTestApp()
	app.adminStore.admins = nil
	app.adminRepo.CreateFirstFunc = func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		return nil, domain.ErrSetupCompleted
	}
	w := app.request("POST", "/v1/auth/setup", map[string]string{"email": "first@firereach.test", "password": "a-long-password"})
	assertError(t, w, http.StatusForbidden, "setup already completed — admin exists")
}

func TestSetup_BadBodyOnAFreshSystem(t *testing.T) {
	app := newTestApp()
	app.adminStore.admins = nil
	w := app.request("POST", "/v1/auth/setup", map[string]string{"email": "not-an-email"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSetup_CountFails(t *testing.T) {
	app := newTestApp()
	app.adminRepo.CountFunc = func(ctx context.Context) (int, error) { return 0, errDatabaseDown }
	w := app.request("POST", "/v1/auth/setup", map[string]string{"email": "first@firereach.test", "password": "a-long-password"})
	assertError(t, w, http.StatusInternalServerError, "failed to check admin count")
}

func TestSetup_DatabaseDownOnInsert(t *testing.T) {
	app := newTestApp()
	app.adminStore.admins = nil
	app.adminRepo.CreateFirstFunc = func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		return nil, errDatabaseDown
	}
	w := app.request("POST", "/v1/auth/setup", map[string]string{"email": "first@firereach.test", "password": "a-long-password"})
	assertError(t, w, http.StatusInternalServerError, "internal server error")
}
