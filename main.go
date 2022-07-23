package main

import (
	"os"
	"strconv"

	"github.com/matheuscscp/splitwiser/bot"
	"github.com/sirupsen/logrus"
)

func main() {
	// telegram inputs
	telegramToken := os.Getenv("BOT_TOKEN")
	telegramChatID, err := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing CHAT_ID env to int64: %v", err)
	}

	// splitwise inputs
	splitwiseToken := os.Getenv("SPLITWISE_TOKEN")
	splitwiseGroupID, err := strconv.ParseInt(os.Getenv("GROUP_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing GROUP_ID env to int64: %v", err)
	}
	splitwiseAnaID, err := strconv.ParseInt(os.Getenv("ANA_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing ANA_ID env to int64: %v", err)
	}
	splitwiseMatheusID, err := strconv.ParseInt(os.Getenv("MATHEUS_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing MATHEUS_ID env to int64: %v", err)
	}

	bot.Run(
		telegramToken, telegramChatID,
		splitwiseToken, splitwiseGroupID, splitwiseAnaID, splitwiseMatheusID,
	)
}
