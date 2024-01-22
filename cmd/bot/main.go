package main

import (
	"context"

	_ "github.com/matheuscscp/splitwiser/cmd"
	"github.com/matheuscscp/splitwiser/internal/bot"
	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/models"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := bot.Run(context.Background(), models.Matheus); err != nil {
		logrus.Fatalf("error running bot: %v", err)
	}
}
