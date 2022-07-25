package main

import (
	"os"

	"github.com/matheuscscp/splitwiser/bot"

	"github.com/sirupsen/logrus"
)

func main() {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "gcloud.json")
	if err := bot.Run("dev" /*nonce*/); err != nil {
		logrus.Fatal(err)
	}
}
