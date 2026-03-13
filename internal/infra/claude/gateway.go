package claude

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/firereach/api/internal/domain"
)

const systemPrompt = `You are FireReach, a fire safety and emergency preparedness assistant for FireReach Ghana.

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
- a burn injury
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

## Response Style
Responses must be:
- concise
- clear
- step-by-step
- practical

Avoid long explanations during emergency scenarios.

## Safety Constraints
Do not:
- speculate
- invent emergency procedures
- give dangerous advice
- provide medical guidance beyond basic burn first aid

If unsure, say:
"I'm not certain. Please contact emergency services or a trained professional."

## Priority Rule
In emergency scenarios:
Safety instructions > evacuation guidance > first aid > explanation.

Life safety always comes first.`

var _ domain.AIGateway = (*Gateway)(nil)

type Gateway struct {
	client anthropic.Client
}

func NewGateway(apiKey string) *Gateway {
	return &Gateway{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

func (g *Gateway) Ask(ctx context.Context, question, topic string) (string, error) {
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
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	})

	if err != nil {
		return "", fmt.Errorf("claude gateway: %w", err)
	}

	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("claude gateway: no text in response")
}
