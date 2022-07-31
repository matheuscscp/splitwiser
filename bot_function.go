package splitwiser

import (
	"context"

	"github.com/matheuscscp/splitwiser/bot"
)

// Bot is a Pub/Sub Cloud Function.
func Bot(ctx context.Context, m PubSubMessage) error {
	return bot.Run()
}
