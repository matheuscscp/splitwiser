package main

import (
	"os"
	"strconv"

	"github.com/matheuscscp/splitwiser/bot"

	"github.com/sirupsen/logrus"
)

func main() {
	conf := &bot.Config{PubSubMessageID: "dev"}
	var err error

	// telegram inputs
	conf.Telegram.Token = os.Getenv("BOT_TOKEN")
	conf.Telegram.ChatID, err = strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing CHAT_ID env to int64: %v", err)
	}

	// splitwise inputs
	conf.Splitwise.Token = os.Getenv("SPLITWISE_TOKEN")
	conf.Splitwise.GroupID, err = strconv.ParseInt(os.Getenv("GROUP_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing GROUP_ID env to int64: %v", err)
	}
	conf.Splitwise.AnaID, err = strconv.ParseInt(os.Getenv("ANA_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing ANA_ID env to int64: %v", err)
	}
	conf.Splitwise.MatheusID, err = strconv.ParseInt(os.Getenv("MATHEUS_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing MATHEUS_ID env to int64: %v", err)
	}

	bot.Run(conf)
}
