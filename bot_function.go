package splitwiser

import (
	"context"

	"github.com/matheuscscp/splitwiser/bot"
)

type (
	// PubSubMessage is the payload of a Pub/Sub event.
	// See the documentation for more details:
	// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
	PubSubMessage struct {
		Data []byte `json:"data"`
	}
)

// Bot is a Pub/Sub Cloud Function.
func Bot(ctx context.Context, m PubSubMessage) error {
	return bot.Run(string(m.Data) /*nonce*/)
}
