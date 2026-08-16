package domain

import (
	"context"
	"fmt"
	"strings"
)

// ResponseKind classifies an AI answer so the client can render the right
// component for the question type. The model chooses the kind; the app owns
// every phone number and call-to-action, so no kind carries a number — that
// is the invariant that stops a hallucinated digit from ever reaching a dial
// control.
type ResponseKind string

const (
	// KindText is a plain prose answer, and the fail-safe every unknown or
	// malformed payload degrades to.
	KindText ResponseKind = "text"
	// KindEmergency is an active emergency (fire, smoke, someone trapped, gas
	// leak): heading, a bold lead, bulleted critical actions, and a prominent
	// call control.
	KindEmergency ResponseKind = "emergency"
	// KindSteps is a how-to / first-aid procedure: a lead line followed by
	// numbered step cards.
	KindSteps ResponseKind = "steps"
	// KindEmergencyNumber answers "what is the emergency number": a label and
	// the number, rendered large by the client, with a call control.
	KindEmergencyNumber ResponseKind = "emergency_number"
	// KindWarning is an answer that carries a hazard caveat: prose plus an
	// amber safety-warning callout.
	KindWarning ResponseKind = "warning"
	// KindOutOfScope is the polite refusal for a non-fire-safety question.
	KindOutOfScope ResponseKind = "out_of_scope"
)

// AIStep is one item in a list answer. A set Title renders as a numbered
// step card (how-to); an empty Title renders as a bullet (emergency/warning).
type AIStep struct {
	Title string
	Body  string
}

// AIResponse is the structured answer the gateway returns. Body is always
// populated; the remaining fields are kind-dependent. Anything the client
// does not recognise must render as Body alone — the fail-safe that keeps
// unvalidated model output safe, the same principle as the guide-body parser.
type AIResponse struct {
	Kind    ResponseKind
	Title   string
	Body    string
	Items   []AIStep
	Ordered bool
	Warning string
}

// Flatten renders the structured answer as plain text. It backs the wire's
// backward-compatible `answer` field and gives the adversarial suite a single
// string to run its substring checks against, so a structured response is
// still checked as prose.
func (r AIResponse) Flatten() string {
	var b strings.Builder
	if r.Title != "" {
		b.WriteString(r.Title)
		b.WriteString("\n\n")
	}
	b.WriteString(r.Body)
	for i, it := range r.Items {
		b.WriteString("\n")
		if r.Ordered {
			fmt.Fprintf(&b, "%d. ", i+1)
		} else {
			b.WriteString("- ")
		}
		if it.Title != "" {
			b.WriteString(it.Title)
			b.WriteString(": ")
		}
		b.WriteString(it.Body)
	}
	if r.Warning != "" {
		b.WriteString("\n\n")
		b.WriteString(r.Warning)
	}
	return b.String()
}

// Turn is one prior message in the conversation, user or assistant. History
// lets a follow-up question carry context; an assistant turn's Content is the
// flattened text of a prior structured answer.
type Turn struct {
	Role    string // "user" or "assistant"
	Content string
}

type AIGateway interface {
	Ask(ctx context.Context, question, topic string, history []Turn) (AIResponse, error)
}
