package main

import (
	"os"

	"github.com/matheuscscp/splitwiser/bot"

	"github.com/sirupsen/logrus"
)

func main() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "gcloud.json")
	if err := bot.Run(); err != nil {
		logrus.Fatal(err)
	}
}
