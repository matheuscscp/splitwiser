package main

import (
	_ "github.com/matheuscscp/splitwiser/cmd"
	"github.com/matheuscscp/splitwiser/internal/bot"
	_ "github.com/matheuscscp/splitwiser/logging"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := bot.Run(); err != nil {
		logrus.Fatalf("error running bot: %v", err)
	}
}
