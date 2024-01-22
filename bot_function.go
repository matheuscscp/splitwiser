package splitwiser

import (
	"context"
	"strings"

	"github.com/matheuscscp/splitwiser/internal/bot"
	"github.com/matheuscscp/splitwiser/models"
)

// Bot is a Pub/Sub Cloud Function.
func Bot(ctx context.Context, m PubSubMessage) error {
	user := strings.Split(string(m.Data), "-")[1]
	return bot.Run(ctx, models.ReceiptItemOwner(user))
}
