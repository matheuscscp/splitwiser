package main

import (
	"os"

	"github.com/matheuscscp/splitwiser/bot"
	_ "github.com/matheuscscp/splitwiser/logging"

	"github.com/sirupsen/logrus"
)

func main() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "gcloud.json")
	if err := bot.Run(); err != nil {
		logrus.Fatalf("error running bot: %v", err)
	}
}
