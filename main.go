package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/matheuscscp/splitwiser/models"
	"github.com/sirupsen/logrus"
)

type (
	botAPI struct {
		*tgbotapi.BotAPI

		chatID int64
	}

	botState         int
	receiptItemOwner string
)

const (
	botStateIdle botState = iota
	botStateParsingReceiptInteractively
	botStateWaitingForPayer
	botStateWaitingForStore

	ana            receiptItemOwner = "a"
	matheus        receiptItemOwner = "m"
	shared         receiptItemOwner = "s"
	delay          receiptItemOwner = "d"
	notReceiptItem receiptItemOwner = "n"

	zeroCents = models.PriceInCents(0)
)

func main() {
	telegramClient, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		logrus.Fatalf("error creating Telegram Bot API client: %v", err)
	}
	logrus.Infof("Authorized on account %s", telegramClient.Self.UserName)

	chatID, err := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing CHAT_ID env to int64: %v", err)
	}

	groupID, err := strconv.ParseInt(os.Getenv("GROUP_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing GROUP_ID env to int64: %v", err)
	}

	anaID, err := strconv.ParseInt(os.Getenv("ANA_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing ANA_ID env to int64: %v", err)
	}

	matheusID, err := strconv.ParseInt(os.Getenv("MATHEUS_ID"), 10, 64)
	if err != nil {
		logrus.Fatalf("error parsing MATHEUS_ID env to int64: %v", err)
	}

	bot := &botAPI{
		BotAPI: telegramClient,
		chatID: chatID,
	}

	// bot state
	botState := botStateIdle
	var receiptItems []*models.ReceiptItem
	totalInCents := make(map[receiptItemOwner]models.PriceInCents)
	var payer receiptItemOwner
	var store string

	resetState := func() {
		botState = botStateIdle
		receiptItems = nil
		totalInCents = make(map[receiptItemOwner]models.PriceInCents)
		payer = ""
		store = ""
	}

	printNextReceiptItem := func() {
		item := receiptItems[0]
		bot.Send(`Item: "%s" (%s)

(see if the next line rings a bell: "%s")

Please choose the owner:
%s - Ana
%s - Matheus
%s - Shared
%s - Put item in the end of the list
%s - Not a receipt item`,
			item.MainLine,
			item.PriceInCents.Format(),
			item.NextLine,
			ana,
			matheus,
			shared,
			delay,
			notReceiptItem,
		)
	}

	updateSplitwise := func() {
		cost := totalInCents[ana]
		payerID, borrowerID := matheusID, anaID
		description := "vegan"
		if payer == ana {
			cost = totalInCents[matheus]
			payerID, borrowerID = anaID, matheusID
			description = "non-vegan"
		}
		bot.Send("Creating non-shared expense...")
		createExpense(
			bot,
			groupID,
			store,
			description,
			cost,
			&userShare{
				userID: payerID,
				paid:   cost,
				owed:   zeroCents,
			},
			&userShare{
				userID: borrowerID,
				paid:   zeroCents,
				owed:   cost,
			},
		)

		costShared := totalInCents[shared]
		halfCostSharedRoundedDown := costShared / 2
		halfCostSharedRoundedUp := (costShared + 1) / 2
		description = "shared"
		bot.Send("Creating shared expense...")
		createExpense(
			bot,
			groupID,
			store,
			description,
			costShared,
			&userShare{
				userID: payerID,
				paid:   costShared,
				owed:   halfCostSharedRoundedDown,
			},
			&userShare{
				userID: borrowerID,
				paid:   zeroCents,
				owed:   halfCostSharedRoundedUp,
			},
		)
	}

	updatesConfig := tgbotapi.NewUpdate(0 /*offset*/)
	updatesConfig.Timeout = 60
	t0 := time.Now()
	for update := range bot.GetUpdatesChan(updatesConfig) {
		if update.Message == nil || update.Message.Chat.ID != chatID {
			continue
		}
		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)

		if msg == "/uptime" {
			bot.Send("I'm up for %s.", time.Since(t0))
			continue
		}

		if botState != botStateIdle && msg == "/abort" {
			resetState()
			bot.Send("Cya.")
			continue
		}

		switch botState {
		case botStateIdle:
			receiptItems = models.ParseReceipt(msg)
			if len(receiptItems) > 0 {
				printNextReceiptItem()
				botState = botStateParsingReceiptInteractively
			} else {
				bot.Send("Could not parse the receipt.")
			}
		case botStateParsingReceiptInteractively:
			owner := receiptItemOwner(msg)
			if owner != ana &&
				owner != matheus &&
				owner != shared &&
				owner != delay &&
				owner != notReceiptItem {
				bot.Send("Invalid choice. Choose one of {%s, %s, %s, %s, %s}.", ana, matheus, shared, delay, notReceiptItem)
			} else {
				// process owner input
				if item := receiptItems[0]; owner == delay {
					receiptItems = append(receiptItems, item)
				} else if owner != notReceiptItem {
					totalInCents[owner] += item.PriceInCents
				}

				// pop item
				receiptItems = receiptItems[1:]

				if len(receiptItems) > 0 {
					printNextReceiptItem()
				} else {
					bot.Send(`Ana's total: %s
Matheus's total: %s
Shared total: %s

Please choose the payer:
%s - Ana
%s - Matheus`,
						totalInCents[ana].Format(),
						totalInCents[matheus].Format(),
						totalInCents[shared].Format(),
						ana,
						matheus,
					)
					botState = botStateWaitingForPayer
				}
			}
		case botStateWaitingForPayer:
			payer = receiptItemOwner(msg)
			if payer != ana && payer != matheus {
				bot.Send("Invalid choice. Choose one of {%s, %s}.", ana, matheus)
			} else {
				bot.Send("Please type in the name of the store.")
				botState = botStateWaitingForStore
			}
		case botStateWaitingForStore:
			store = msg
			if len(store) == 0 {
				bot.Send("Store name cannot be empty.")
			} else {
				updateSplitwise()
				resetState()
			}
		default:
			logrus.Errorf("invalid botState: %d", botState)
		}
	}
}

func (b *botAPI) Send(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	msg := tgbotapi.NewMessage(b.chatID, s)
	if _, err := b.BotAPI.Send(msg); err != nil {
		logrus.Errorf("error sending message '%s': %v", s, err)
	} else {
		logrus.Infof("[%s] %s", b.Self.UserName, s)
	}
}
