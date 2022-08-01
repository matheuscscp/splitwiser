package main

import (
	"github.com/matheuscscp/splitwiser/bot"
	_ "github.com/matheuscscp/splitwiser/cmd"
	_ "github.com/matheuscscp/splitwiser/logging"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := bot.Run(); err != nil {
		logrus.Fatalf("error running bot: %v", err)
	}
}
