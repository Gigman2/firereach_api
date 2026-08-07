package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/firereach/api/internal/adapter/dto"
)

func TestErrorResponse_MarshalsToErrorKey(t *testing.T) {
	blob, err := json.Marshal(dto.ErrorResponse{Error: "station not found"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got, want := string(blob), `{"error":"station not found"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestMessageResponse_MarshalsToMessageKey(t *testing.T) {
	blob, err := json.Marshal(dto.MessageResponse{Message: "station updated"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got, want := string(blob), `{"message":"station updated"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestCreatedAdminResponse_MarshalsToIDAndEmail(t *testing.T) {
	blob, err := json.Marshal(dto.CreatedAdminResponse{
		ID:    "11111111-1111-1111-1111-111111111111",
		Email: "admin@firereach.gh",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	want := `{"id":"11111111-1111-1111-1111-111111111111","email":"admin@firereach.gh"}`
	if got := string(blob); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLoginResponse_MarshalsToTokenKey(t *testing.T) {
	blob, err := json.Marshal(dto.LoginResponse{Token: "abc.def.ghi"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got, want := string(blob), `{"token":"abc.def.ghi"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLoginRequest_UnmarshalsEmailAndPassword(t *testing.T) {
	var req dto.LoginRequest
	if err := json.Unmarshal([]byte(`{"email":"a@b.c","password":"secret123"}`), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Email != "a@b.c" || req.Password != "secret123" {
		t.Errorf("got %+v", req)
	}
}

func TestRegisterRequest_UnmarshalsEmailAndPassword(t *testing.T) {
	var req dto.RegisterRequest
	if err := json.Unmarshal([]byte(`{"email":"a@b.c","password":"secret123"}`), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Email != "a@b.c" || req.Password != "secret123" {
		t.Errorf("got %+v", req)
	}
}
