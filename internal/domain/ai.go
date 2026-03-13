package domain

import "context"

type AIGateway interface {
	Ask(ctx context.Context, question, topic string) (string, error)
}
