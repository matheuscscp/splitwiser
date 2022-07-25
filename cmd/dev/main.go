package main

import (
	"context"
	"os"

	"github.com/matheuscscp/splitwiser"
)

func main() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "gcloud.json")
	splitwiser.Bot(context.Background(), splitwiser.PubSubMessage{
		Data: []byte("dev"),
	})
}
