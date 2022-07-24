package main

import (
	"context"
	"os"

	"github.com/matheuscscp/splitwiser"
)

func main() {
	os.Setenv(splitwiser.ConfFileEnv, "config.yml")
	splitwiser.Bot(context.Background(), splitwiser.PubSubMessage{
		Data:      []byte("start"),
		MessageID: []byte("dev"),
	})
}
