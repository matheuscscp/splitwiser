package bot

import (
	"fmt"
	"time"

	"github.com/matheuscscp/splitwiser/models"
	"github.com/matheuscscp/splitwiser/splitwise"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type (
	botAPI struct {
		*tgbotapi.BotAPI

		chatID int64
		closed bool
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

	botTimeout              = 10 * time.Minute
	botTimeoutWatchInterval = 10 * time.Second
)

func (b *botAPI) send(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	msg := tgbotapi.NewMessage(b.chatID, s)
	if _, err := b.BotAPI.Send(msg); err != nil {
		logrus.Errorf("error sending message '%s': %v", s, err)
	} else {
		logrus.Infof("[%s] %s", b.Self.UserName, s)
	}
}

func (b *botAPI) close() {
	if b.closed {
		return
	}
	b.closed = true
	b.StopReceivingUpdates()
}

// Run starts the bot and returns when the bot has finished processing all receipts.
func Run(
	telegramToken string, telegramChatID int64,
	splitwiseToken string, splitwiseGroupID, splitwiseAnaID, splitwiseMatheusID int64,
) {
	telegramClient, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		logrus.Fatalf("error creating Telegram Bot API client: %v", err)
	}
	logrus.Infof("Authorized on Telegram bot account %s", telegramClient.Self.UserName)

	bot := &botAPI{
		BotAPI: telegramClient,
		chatID: telegramChatID,
	}

	// timeout thread
	lastActivity := time.Now()
	timeoutThreadShutdownChannel := make(chan struct{})
	go func() {
		timer := time.NewTimer(botTimeoutWatchInterval)
		for {
			select {
			case <-timer.C:
				if botTimeout <= time.Since(lastActivity) {
					bot.close()
					return
				}
				timer.Reset(botTimeoutWatchInterval)
			case <-timeoutThreadShutdownChannel:
				timer.Stop()
				return
			}
		}
	}()

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
		bot.send(`Item: "%s" (%s)

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
		payerID, borrowerID := splitwiseMatheusID, splitwiseAnaID
		description := "vegan"
		if payer == ana {
			cost = totalInCents[matheus]
			payerID, borrowerID = splitwiseAnaID, splitwiseMatheusID
			description = "non-vegan"
		}
		bot.send("Creating non-shared expense...")
		expenseMsg := splitwise.CreateExpense(
			splitwiseToken,
			splitwiseGroupID,
			store,
			description,
			cost,
			&splitwise.UserShare{
				UserID: payerID,
				Paid:   cost,
				Owed:   zeroCents,
			},
			&splitwise.UserShare{
				UserID: borrowerID,
				Paid:   zeroCents,
				Owed:   cost,
			},
		)
		bot.send(expenseMsg)

		costShared := totalInCents[shared]
		halfCostSharedRoundedDown := costShared / 2
		halfCostSharedRoundedUp := (costShared + 1) / 2
		description = "shared"
		bot.send("Creating shared expense...")
		expenseMsg = splitwise.CreateExpense(
			splitwiseToken,
			splitwiseGroupID,
			store,
			description,
			costShared,
			&splitwise.UserShare{
				UserID: payerID,
				Paid:   costShared,
				Owed:   halfCostSharedRoundedDown,
			},
			&splitwise.UserShare{
				UserID: borrowerID,
				Paid:   zeroCents,
				Owed:   halfCostSharedRoundedUp,
			},
		)
		bot.send(expenseMsg)
	}

	updatesConfig := tgbotapi.NewUpdate(0 /*offset*/)
	updatesConfig.Timeout = 60
	t0 := time.Now()
	for update := range bot.GetUpdatesChan(updatesConfig) {
		if bot.closed || update.Message == nil || update.Message.Chat.ID != telegramChatID {
			continue
		}
		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)
		lastActivity = time.Now()

		if botState != botStateIdle && msg == "/abort" {
			resetState()
			bot.send("Okay, start a new receipt then.")
			continue
		}

		if msg == "/uptime" {
			bot.send("I'm up for %s.", time.Since(t0))
			continue
		}

		if msg == "/finish" {
			bot.send("Okay, I will shutdown.")
			close(timeoutThreadShutdownChannel)
			bot.close()
			continue
		}

		switch botState {
		case botStateIdle:
			receiptItems = models.ParseReceipt(msg)
			if len(receiptItems) > 0 {
				printNextReceiptItem()
				botState = botStateParsingReceiptInteractively
			} else {
				bot.send("Could not parse the receipt.")
			}
		case botStateParsingReceiptInteractively:
			owner := receiptItemOwner(msg)
			if owner != ana &&
				owner != matheus &&
				owner != shared &&
				owner != delay &&
				owner != notReceiptItem {
				bot.send("Invalid choice. Choose one of {%s, %s, %s, %s, %s}.", ana, matheus, shared, delay, notReceiptItem)
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
					bot.send(`Ana's total: %s
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
				bot.send("Invalid choice. Choose one of {%s, %s}.", ana, matheus)
			} else {
				bot.send("Please type in the name of the store.")
				botState = botStateWaitingForStore
			}
		case botStateWaitingForStore:
			store = msg
			if len(store) == 0 {
				bot.send("Store name cannot be empty.")
			} else {
				updateSplitwise()
				resetState()
			}
		default:
			logrus.Errorf("invalid botState: %d", botState)
		}
	}

	bot.send("Cya.")
}
