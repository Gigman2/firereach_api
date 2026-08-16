package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/firereach/api/internal/domain"
)

// EmergencyPhone is Ghana's fire and emergency services number, by explicit
// project-owner decision. It is referenced by systemPrompt below and reused
// by the adversarial test suite so the number lives in exactly one place.
const EmergencyPhone = "192"

var systemPrompt = fmt.Sprintf(`You are FireReach, a fire safety and emergency preparedness assistant for FireReach Ghana.

Your purpose is to provide reliable information about:
- Fire hazards
- Fire prevention
- Fire safety education
- Burn first aid
- Emergency evacuation procedures

## Scope Rules
Only answer questions related to fire safety, fire hazards, burn first aid, and emergency response.

If a user asks about unrelated topics, politely respond:
"I can only assist with fire safety and emergency preparedness questions."

## Emergency Detection
If the user describes:
- active fire
- smoke in a building
- someone trapped
- a severe burn (see High-Risk Burns below)
- gas leak or explosion risk

Treat the situation as a **possible emergency**.

## Emergency Response Protocol
When an emergency is detected:

1. Provide the most critical safety action first.
2. Give clear evacuation or containment steps.
3. Provide immediate first aid instructions if relevant.
4. Strongly advise contacting emergency services.

Example:
"Call the Ghana National Fire Service immediately."

The High-Risk Burns case below (see Safety Constraints) replaces this
four-step protocol with its own three-part response — it does not layer on
top of it.

## Response Style
Responses must be:
- concise
- clear
- step-by-step
- practical

Avoid long explanations during emergency scenarios. The High-Risk Burns case
below is the exception to step-by-step: three plain sentences, not a list.

## Safety Constraints
Do not:
- speculate
- invent emergency procedures
- give dangerous advice
- provide medical guidance beyond basic burn first aid

If unsure, say:
"I'm not certain. Please contact emergency services or a trained professional."

Never name a medication, brand, or dosage — not even an over-the-counter one,
and not even while explaining that you cannot discuss it. This includes brand
names such as Panadol, Brufen, Nurofen, Advil, Tylenol, Ibucap, and Efpac, and
any paracetamol- or ibuprofen-branded product. Do not repeat back the name of
whatever medication the user asked about, not even to decline it — naming it
in a refusal is still naming it. If asked about pain relief, respond only
with:
"I can't advise on medication, including its name. A pharmacist or clinician
can help with pain relief. For the burn itself, I can help with fire safety
first aid."
Then continue with fire safety or burn first aid only if relevant.

## Minor Burns: First Aid, Not 192
A small, superficial thermal burn or scald — hot water, hot oil, a brief
touch — on an adult or older child, that is not deep, not large, and not on
the face or airway, is first aid, not a 192 emergency. Use kind=steps and
give the burn first-aid steps; do not tell them to call 192 for this.

End by saying when to get more help, because some burns that look minor are
not: a burn on a hand, foot, or joint, one that goes around a finger or
limb, or one that blisters badly should be seen at a pharmacy or clinic.
Treat it as an emergency instead (High-Risk Burns) when it is chemical or
electrical, on a baby or small child, on the face or airway, covers a large
area, is deep, or there is any trouble breathing.

## High-Risk Burns: Safety Action, Then Defer
For a chemical burn, an electrical burn, a burn to a child, a burn to the
face or airway, a deep burn, or any injury covering a large area: give exactly
three things and nothing else.

1. The single scene-safety action that stops further harm right now:
   - Electrical burn: do not touch them until the power is off.
   - Chemical burn: get the chemical off — brush off a dry chemical, then
     start rinsing with water now.
   - Burn to a child, to the face, hands, or airway, or a large-area burn:
     get them away from the source of the burn and keep them still.
2. That this needs emergency medical help now, and the number: %s.
3. That the Burns guide in this app has the steps to follow while help is on
   the way.

Nothing else. No cooling duration, no rinsing duration, no dressings, no
blister management, no monitoring schedule, and no numbered or bulleted list
in your reply — those live in the app's reviewed guide, not in a generated
response. The scene-safety action above is the one hands-on instruction you
give; do not extend it into an ongoing treatment, and do not attach a time or
duration to it — say only to start it now ("start rinsing now," never "rinse
for 10 minutes" or "rinse for at least 15 minutes"). The guide has the
timing.

## Priority Rule
In emergency scenarios:
Safety instructions > evacuation guidance > first aid > explanation.

Life safety always comes first.

## Response Format
Always answer by calling the respond_safety tool; never reply in free text.
Choose the kind that matches the question, put the prose in body, and put any
list into items.

- emergency: an active fire, smoke, someone trapped, a severe burn (the
  High-Risk Burns case), or a gas leak. A minor burn is not an emergency; use
  steps for it. Set title to "ACTIVE EMERGENCY". Put the single most critical action
  first in body. Put the remaining critical actions into items as short
  bullets, leaving each item title empty and the action in its body. Set
  ordered=false.
- steps: a how-to or first-aid procedure. Put the lead line in body and one
  entry per step in items, each with a short title and a body. Set
  ordered=true.
- emergency_number: the user asked for the emergency number. Set title to
  "Fire Emergency Number" and put a short lead in body, such as "The Ghana
  National Fire Service emergency number is:". Do not write the digits; the
  app displays them.
- warning: the answer carries a hazard caveat. Put the answer in body, any
  bullets in items, and the safety-warning callout in warning.
- out_of_scope: not a fire-safety question. Put the polite refusal in body.
- text: anything else. Put the answer in body and leave items empty.

The High-Risk Burns case stays prose: use kind=emergency, put the three
sentences in body, and leave items empty.

The app renders the call control and the emergency number itself, so the
emergency and emergency_number cards never need the digits in your text.`, EmergencyPhone)

// medicationBrands is the same brand/generic list systemPrompt above already
// forbids the model from naming (kept in sync with the brands list in
// gateway_adversarial_test.go's TestRefusesToNameMedication). The prompt is
// prose sampled probabilistically; the ledger records the medication case as
// model-sampling-sensitive, meaning it has been observed failing
// intermittently even with this exact prompt. This slice backs a
// deterministic filter applied to what the model actually returned, so a
// sampling miss on the prompt still cannot reach the user.
var medicationBrands = []string{
	"ibuprofen", "paracetamol", "acetaminophen", "aspirin",
	"panadol", "brufen", "nurofen", "advil", "tylenol", "ibucap", "efpac",
}

// medicationDosagePattern catches a numeric dosage in milligrams (e.g.
// "400mg", "400 mg") regardless of which drug it's attached to — a response
// can leak a dose without ever naming a brand on the medicationBrands list
// above. Named distinctly from gateway_adversarial_test.go's own
// dosagePattern (same regex, same package, different build tag) to avoid a
// redeclaration when both are compiled together with `-tags=adversarial`.
var medicationDosagePattern = regexp.MustCompile(`(?i)\b[0-9]{1,4}\s*mg\b`)

// medicationRefusal mirrors the refusal systemPrompt already instructs the
// model to give for a pain-relief question, so a filtered response reads the
// same as a correctly-following one rather than as a visibly different
// fallback.
//
// This substitution discards the model's entire answer, not just the
// medication mention — so if the question described a real emergency (a
// burn injury, say), the scene-safety action and the advice to call for
// help go with it. The refusal must stand on its own rather than leave the
// user with nothing but a pharmacist referral, so it ends by pointing at the
// same emergency number systemPrompt above already gives for every other
// emergency case.
const medicationRefusal = "I can't advise on medication, including its name. A pharmacist or clinician can help with pain relief. For the burn itself, I can help with fire safety first aid. If this is an emergency, call " + EmergencyPhone + " now."

// containsMedication reports whether text names a forbidden medication brand
// or generic name, or states a dosage in milligrams. Belt and braces: the
// system prompt already forbids both, but this is a deterministic check
// applied to the model's actual output, not a hope that sampling followed
// the prompt this time.
func containsMedication(text string) bool {
	low := strings.ToLower(text)
	for _, brand := range medicationBrands {
		if strings.Contains(low, brand) {
			return true
		}
	}
	return medicationDosagePattern.MatchString(text)
}

var _ domain.AIGateway = (*Gateway)(nil)

type Gateway struct {
	client anthropic.Client
}

func NewGateway(apiKey string) *Gateway {
	return &Gateway{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

// respondSafetyTool is the single tool the model is forced to call, which is
// what makes its answer structured JSON instead of free prose. The schema has
// no phone-number field on purpose: every number and call-to-action is added
// by the app, so a hallucinated digit can never reach a dial control.
var respondSafetyTool = anthropic.ToolUnionParam{
	OfTool: &anthropic.ToolParam{
		Name: "respond_safety",
		Description: anthropic.String(
			"Report the answer to the user's fire-safety question. Choose the kind that " +
				"matches the question, put the lead prose in body, and put any list into items."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"kind": map[string]any{
					"type": "string",
					"enum": []string{"text", "emergency", "steps", "emergency_number", "warning", "out_of_scope"},
					"description": "text: a plain answer. emergency: an active fire, smoke, someone trapped, a burn injury, or a gas leak. steps: a how-to or first-aid procedure. emergency_number: the user asked for the emergency number. warning: the answer carries a hazard caveat. out_of_scope: not a fire-safety question.",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Short heading for emergency and emergency_number cards. Empty otherwise.",
				},
				"body": map[string]any{
					"type":        "string",
					"description": "The lead prose of the answer. Always required.",
				},
				"items": map[string]any{
					"type":        "array",
					"description": "The list part of the answer. For steps, one entry per numbered step with a short title and body. For emergency and warning, one entry per bullet with an empty title.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"title": map[string]any{"type": "string"},
							"body":  map[string]any{"type": "string"},
						},
						"required": []string{"body"},
					},
				},
				"ordered": map[string]any{
					"type":        "boolean",
					"description": "True when items are an ordered procedure (steps); false for unordered bullets.",
				},
				"warning": map[string]any{
					"type":        "string",
					"description": "For kind=warning, the safety-warning callout text. Empty otherwise.",
				},
			},
			Required: []string{"kind", "body"},
		},
	},
}

// respondSafetyInput is the wire shape of the respond_safety tool call.
type respondSafetyInput struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Items []struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"items"`
	Ordered bool   `json:"ordered"`
	Warning string `json:"warning"`
}

func (g *Gateway) Ask(ctx context.Context, question, topic string, history []domain.Turn) (domain.AIResponse, error) {
	userMsg := question
	if topic != "" {
		userMsg = fmt.Sprintf("[Topic: %s] %s", topic, question)
	}

	resp, err := g.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Tools: []anthropic.ToolUnionParam{respondSafetyTool},
		// Forcing the tool is what makes the output structured: the model
		// cannot answer in free prose, only by filling the schema.
		ToolChoice: anthropic.ToolChoiceParamOfTool("respond_safety"),
		Messages:   buildMessages(history, userMsg),
	})

	if err != nil {
		return domain.AIResponse{}, fmt.Errorf("claude gateway: %w", err)
	}

	return parseResponse(resp)
}

// maxHistoryTurns bounds how much of a long chat is sent to the model, so a
// lengthy conversation cannot blow the token budget. The most recent turns
// carry the context a follow-up needs; older ones are dropped first.
const maxHistoryTurns = 12

// buildMessages assembles the conversation for the API: the prior turns plus
// the new question as the final user turn. The API requires alternating roles
// starting with a user message, so consecutive same-role turns are merged and
// any leading assistant turn (a greeting with no question before it) is
// dropped. Assistant turns carry the flattened text of a prior structured
// answer, never a tool_use block, which keeps the history a plain
// user/assistant alternation the forced tool call does not disturb.
func buildMessages(history []domain.Turn, userMsg string) []anthropic.MessageParam {
	var turns []domain.Turn
	add := func(role, content string) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		if role != "assistant" {
			role = "user"
		}
		if len(turns) > 0 && turns[len(turns)-1].Role == role {
			turns[len(turns)-1].Content += "\n\n" + content
			return
		}
		turns = append(turns, domain.Turn{Role: role, Content: content})
	}
	for _, t := range history {
		add(t.Role, t.Content)
	}
	add("user", userMsg)

	for len(turns) > 0 && turns[0].Role == "assistant" {
		turns = turns[1:]
	}
	if len(turns) > maxHistoryTurns {
		turns = turns[len(turns)-maxHistoryTurns:]
		for len(turns) > 0 && turns[0].Role == "assistant" {
			turns = turns[1:]
		}
	}

	msgs := make([]anthropic.MessageParam, 0, len(turns))
	for _, t := range turns {
		if t.Role == "assistant" {
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(t.Content)))
		} else {
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(t.Content)))
		}
	}
	return msgs
}

// parseResponse extracts the structured answer from the forced tool call and
// fails safe to a plain-text answer on any unexpected shape, so malformed or
// surprising model output never reaches the user as a broken card.
func parseResponse(resp *anthropic.Message) (domain.AIResponse, error) {
	for _, block := range resp.Content {
		if block.Type != "tool_use" || block.Name != "respond_safety" {
			continue
		}
		// Read Input and Name off the union directly rather than AsToolUse(),
		// which re-unmarshals from a raw buffer only populated on a live
		// response — the direct fields are set in both paths.
		var in respondSafetyInput
		if err := json.Unmarshal(block.Input, &in); err == nil {
			return sanitizeResponse(normalize(in)), nil
		}
		// Malformed tool input: fall through to the prose fail-safe below.
	}

	// Fail-safe: any prose the model returned becomes a plain-text answer.
	for _, block := range resp.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return sanitizeResponse(domain.AIResponse{Kind: domain.KindText, Body: block.Text}), nil
		}
	}

	return domain.AIResponse{}, fmt.Errorf("claude gateway: no usable response")
}

// normalize maps the wire shape onto the domain type, whitelisting the kind
// (an unknown kind degrades to plain text) and dropping empty entries, so the
// client only ever has to render clean, known values.
func normalize(in respondSafetyInput) domain.AIResponse {
	kind := domain.ResponseKind(in.Kind)
	switch kind {
	case domain.KindText, domain.KindEmergency, domain.KindSteps,
		domain.KindEmergencyNumber, domain.KindWarning, domain.KindOutOfScope:
	default:
		kind = domain.KindText
	}

	body := strings.TrimSpace(in.Body)
	if body == "" {
		// A structured answer with no prose is not renderable; degrade to the
		// same safe line the system prompt already uses for uncertainty.
		body = "I'm not certain. Please contact emergency services or a trained professional."
	}

	items := make([]domain.AIStep, 0, len(in.Items))
	for _, it := range in.Items {
		b := strings.TrimSpace(it.Body)
		if b == "" {
			continue
		}
		items = append(items, domain.AIStep{Title: strings.TrimSpace(it.Title), Body: b})
	}

	return domain.AIResponse{
		Kind:    kind,
		Title:   strings.TrimSpace(in.Title),
		Body:    body,
		Items:   items,
		Ordered: in.Ordered,
		Warning: strings.TrimSpace(in.Warning),
	}
}

// sanitizeResponse applies the deterministic medication filter to every
// user-visible field of a structured answer, not just one prose string: a
// contaminated step title or warning trips it exactly as a contaminated body
// would. On any hit the whole answer is replaced by the scripted refusal, the
// same substitution the prose path always made.
func sanitizeResponse(r domain.AIResponse) domain.AIResponse {
	fields := []string{r.Title, r.Body, r.Warning}
	for _, it := range r.Items {
		fields = append(fields, it.Title, it.Body)
	}
	for _, f := range fields {
		if containsMedication(f) {
			return domain.AIResponse{Kind: domain.KindText, Body: medicationRefusal}
		}
	}
	return r
}
