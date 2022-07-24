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
		MessageID []byte `json:"messageId"`
	}
)

const (
	ConfFileEnv = "CONF_FILE"
)

// Bot is a Pub/Sub Cloud Function.
func Bot(ctx context.Context, m PubSubMessage) error {
	if s := string(m.Data); s != "start" {
		return fmt.Errorf("message data should be 'start' but was '%s'", s)
	}

	// read config
	b, err := os.ReadFile(os.Getenv(ConfFileEnv))
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}
	var conf bot.Config
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}
	conf.PubSubMessageID = string(m.MessageID)

	bot.Run(&conf)

	return nil
}
