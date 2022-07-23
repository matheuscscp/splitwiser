package splitwiser

import (
	"context"
	"fmt"
	"os"

	"github.com/matheuscscp/splitwiser/bot"

	"gopkg.in/yaml.v3"
)

type (
	// PubSubMessage is the payload of a Pub/Sub event.
	// See the documentation for more details:
	// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
	PubSubMessage struct {
		Data      []byte `json:"data"`
		MessageID string `json:"messageId"`
	}
)

// Bot is a Pub/Sub Cloud Function.
func Bot(ctx context.Context, m PubSubMessage) error {
	if string(m.Data) != "start" {
		return nil
	}

	// read config
	b, err := os.ReadFile("/etc/secrets/config/latest")
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}
	var conf bot.Config
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}
	conf.PubSubMessageID = m.MessageID

	bot.Run(&conf)

	return nil
}
