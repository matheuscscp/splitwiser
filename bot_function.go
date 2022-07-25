package splitwiser

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/matheuscscp/splitwiser/bot"
	"github.com/matheuscscp/splitwiser/config"

	"gopkg.in/yaml.v3"
)

type (
	// PubSubMessage is the payload of a Pub/Sub event.
	// See the documentation for more details:
	// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
	PubSubMessage struct {
		Data []byte `json:"data"`
	}
)

const (
	ConfFileEnv = "CONF_FILE"
)

// Bot is a Pub/Sub Cloud Function.
func Bot(ctx context.Context, m PubSubMessage) error {
	startTime := time.Now()
	nonce := string(m.Data)

	// read config file
	b, err := os.ReadFile(os.Getenv(ConfFileEnv))
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// unmarshal config
	var conf config.Config
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	bot.Run(&conf, startTime, nonce)
	return nil
}
