package claude

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/firereach/api/internal/domain"
)

// toolUseMsg builds a Message carrying a single respond_safety tool_use block
// with the given raw JSON input, the shape a live structured response arrives
// in.
func toolUseMsg(t *testing.T, input string) *anthropic.Message {
	t.Helper()
	return &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			{Type: "tool_use", Name: "respond_safety", Input: json.RawMessage(input)},
		},
	}
}

func TestParseResponse_StructuredSteps(t *testing.T) {
	msg := toolUseMsg(t, `{"kind":"steps","body":"Here is how to treat a minor burn:","items":[{"title":"Cool the burn","body":"Hold it under cool running water."},{"title":"Remove jewelry","body":"Take off rings near the burn."}],"ordered":true}`)

	r, err := parseResponse(msg)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if r.Kind != domain.KindSteps {
		t.Errorf("kind = %q, want steps", r.Kind)
	}
	if !r.Ordered {
		t.Error("ordered = false, want true")
	}
	if len(r.Items) != 2 || r.Items[0].Title != "Cool the burn" {
		t.Errorf("items not parsed: %+v", r.Items)
	}
}

func TestParseResponse_UnknownKindFailsSafeToText(t *testing.T) {
	msg := toolUseMsg(t, `{"kind":"diagnosis","body":"Some answer."}`)

	r, err := parseResponse(msg)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if r.Kind != domain.KindText {
		t.Errorf("kind = %q, want text fail-safe", r.Kind)
	}
	if r.Body != "Some answer." {
		t.Errorf("body = %q", r.Body)
	}
}

func TestParseResponse_MalformedToolInputFallsBackToProse(t *testing.T) {
	// A tool_use block whose input is not valid JSON, followed by a text block:
	// the prose must surface as a plain-text answer, not a broken card.
	msg := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			{Type: "tool_use", Name: "respond_safety", Input: json.RawMessage(`{not json`)},
			{Type: "text", Text: "Evacuate the building now."},
		},
	}

	r, err := parseResponse(msg)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if r.Kind != domain.KindText || r.Body != "Evacuate the building now." {
		t.Errorf("expected prose fail-safe, got %+v", r)
	}
}

func TestParseResponse_EmptyIsError(t *testing.T) {
	_, err := parseResponse(&anthropic.Message{})
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestNormalize_DropsEmptyItemsAndTrims(t *testing.T) {
	var in respondSafetyInput
	in.Kind = "emergency"
	in.Title = "  ACTIVE EMERGENCY  "
	in.Body = "  Get out now. "
	in.Items = []struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{
		{Title: "", Body: "Stay low"},
		{Title: "", Body: "   "}, // dropped: empty body
	}

	r := normalize(in)
	if r.Title != "ACTIVE EMERGENCY" || r.Body != "Get out now." {
		t.Errorf("not trimmed: %+v", r)
	}
	if len(r.Items) != 1 || r.Items[0].Body != "Stay low" {
		t.Errorf("empty item not dropped: %+v", r.Items)
	}
}

func TestSanitizeResponse_MedicationInAnyField(t *testing.T) {
	cases := []struct {
		name string
		resp domain.AIResponse
	}{
		{"body", domain.AIResponse{Kind: domain.KindText, Body: "Take ibuprofen for it."}},
		{"step title", domain.AIResponse{Kind: domain.KindSteps, Body: "ok", Items: []domain.AIStep{{Title: "Panadol", Body: "x"}}}},
		{"step body", domain.AIResponse{Kind: domain.KindSteps, Body: "ok", Items: []domain.AIStep{{Title: "Dose", Body: "Take 400mg now"}}}},
		{"warning", domain.AIResponse{Kind: domain.KindWarning, Body: "ok", Warning: "avoid paracetamol"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeResponse(c.resp)
			if got.Body != medicationRefusal {
				t.Errorf("medication in %s not filtered: %+v", c.name, got)
			}
			if got.Kind != domain.KindText {
				t.Errorf("refusal kind = %q, want text", got.Kind)
			}
		})
	}
}

func TestSanitizeResponse_LeavesCleanAnswerAlone(t *testing.T) {
	clean := domain.AIResponse{
		Kind: domain.KindSteps,
		Body: "Here is how to treat a minor burn:",
		Items: []domain.AIStep{
			{Title: "Cool the burn", Body: "Hold it under cool running water."},
		},
		Ordered: true,
	}
	got := sanitizeResponse(clean)
	if got.Kind != domain.KindSteps || len(got.Items) != 1 || got.Body != clean.Body {
		t.Errorf("clean answer altered: %+v", got)
	}
}

func TestFlatten_RendersOrderedAndUnordered(t *testing.T) {
	steps := domain.AIResponse{
		Kind: domain.KindSteps, Body: "Lead.", Ordered: true,
		Items: []domain.AIStep{{Title: "One", Body: "First"}, {Title: "Two", Body: "Second"}},
	}
	flat := steps.Flatten()
	if !strings.Contains(flat, "1. One: First") || !strings.Contains(flat, "2. Two: Second") {
		t.Errorf("ordered flatten wrong:\n%s", flat)
	}

	em := domain.AIResponse{
		Kind: domain.KindEmergency, Title: "ACTIVE EMERGENCY", Body: "Get out.",
		Items: []domain.AIStep{{Body: "Stay low"}},
	}
	flat = em.Flatten()
	if !strings.HasPrefix(flat, "ACTIVE EMERGENCY") || !strings.Contains(flat, "- Stay low") {
		t.Errorf("emergency flatten wrong:\n%s", flat)
	}
}

// roles extracts the role sequence of a built message list.
func roles(msgs []anthropic.MessageParam) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = string(m.Role)
	}
	return out
}

func TestBuildMessages_NoHistory(t *testing.T) {
	msgs := buildMessages(nil, "My house is on fire")
	if len(msgs) != 1 || string(msgs[0].Role) != "user" {
		t.Fatalf("expected single user message, got %v", roles(msgs))
	}
}

func TestBuildMessages_PreservesAlternation(t *testing.T) {
	h := []domain.Turn{
		{Role: "user", Content: "How do I treat a burn?"},
		{Role: "assistant", Content: "Cool it under water."},
	}
	msgs := buildMessages(h, "What if it blisters?")
	got := roles(msgs)
	want := []string{"user", "assistant", "user"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("roles = %v, want %v", got, want)
	}
}

func TestBuildMessages_MergesConsecutiveSameRole(t *testing.T) {
	h := []domain.Turn{
		{Role: "user", Content: "first"},
		{Role: "user", Content: "second"},
	}
	msgs := buildMessages(h, "third")
	// three consecutive user turns merge into one, and the new question merges
	// with them too.
	if len(msgs) != 1 || string(msgs[0].Role) != "user" {
		t.Fatalf("expected one merged user message, got %v", roles(msgs))
	}
	b, _ := json.Marshal(msgs[0])
	for _, want := range []string{"first", "second", "third"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("merged message missing %q: %s", want, b)
		}
	}
}

func TestBuildMessages_DropsLeadingAssistant(t *testing.T) {
	h := []domain.Turn{
		{Role: "assistant", Content: "Hi! I am the assistant."},
		{Role: "user", Content: "question"},
	}
	msgs := buildMessages(h, "follow up")
	if string(msgs[0].Role) != "user" {
		t.Fatalf("first message must be user, got %v", roles(msgs))
	}
}

func TestBuildMessages_BoundsHistory(t *testing.T) {
	var h []domain.Turn
	for i := 0; i < 40; i++ {
		h = append(h, domain.Turn{Role: "user", Content: "q"})
		h = append(h, domain.Turn{Role: "assistant", Content: "a"})
	}
	msgs := buildMessages(h, "latest")
	if len(msgs) > maxHistoryTurns {
		t.Fatalf("expected at most %d messages, got %d", maxHistoryTurns, len(msgs))
	}
	if string(msgs[len(msgs)-1].Role) != "user" {
		t.Errorf("last message must be the new user question, got %v", roles(msgs))
	}
}
