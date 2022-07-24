package bot

import (
	"errors"
	"fmt"
	"time"

	"github.com/matheuscscp/splitwiser/checkpoint"
	"github.com/matheuscscp/splitwiser/config"
	"github.com/matheuscscp/splitwiser/models"
	"github.com/matheuscscp/splitwiser/splitwise"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})
}

type (
	botAPI struct {
		*tgbotapi.BotAPI
		checkpoint.Manager

		chatID int64
		closed bool
	}

	botState int
)

const (
	botStateIdle botState = iota
	botStateParsingReceiptInteractively
	botStateWaitingForPayer
	botStateWaitingForStore

	botTimeout              = 7 * time.Minute
	botTimeoutWatchInterval = 3 * time.Second

	delayDecision = "d"
	backOneItem   = "b"
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

func (b *botAPI) sendReceiptItem(item *models.ReceiptItem, lastModifiedReceiptItem int) {
	var back string
	if lastModifiedReceiptItem >= 0 {
		back = fmt.Sprintf("\n%s - Back one receipt item", backOneItem)
	}
	b.send(`Item: "%s" (%s)

(see if the next line rings a bell: "%s")

Please choose the owner:
%s - Ana
%s - Matheus
%s - Shared
%s - Not a receipt item
%s - Delay item decision%s`,
		item.MainLine,
		item.Price.Format(),
		item.NextLine,
		models.Ana,
		models.Matheus,
		models.Shared,
		models.NotReceiptItem,
		delayDecision,
		back,
	)
}

func (b *botAPI) sendOwnerChoice(lastModifiedReceiptItem int) {
	var back string
	if lastModifiedReceiptItem >= 0 {
		back = fmt.Sprintf(", %s", backOneItem)
	}
	b.send(
		"Invalid choice. Choose one of {%s, %s, %s, %s, %s%s}.",
		models.Ana, models.Matheus, models.Shared, models.NotReceiptItem, delayDecision, back,
	)
}

func (b *botAPI) sendPayerChoice(receipt models.Receipt) {
	totalInCents := receipt.ComputeTotals()
	b.send(`Ana's total: %s
Matheus's total: %s
Shared total: %s

Please choose the payer:
%s - Ana
%s - Matheus`,
		totalInCents[models.Ana].Format(),
		totalInCents[models.Matheus].Format(),
		totalInCents[models.Shared].Format(),
		models.Ana,
		models.Matheus,
	)
}

func (b *botAPI) storeCheckpoint(receipt models.Receipt) {
	if err := b.Manager.StoreCheckpoint(receipt); err != nil {
		b.send("I had an unexpected error storing the checkpoint: %v", err)
	}
}

func (b *botAPI) deleteCheckpoint() {
	if err := b.Manager.DeleteCheckpoint(); err != nil {
		b.send("I had an unexpected error deleting the checkpoint: %v", err)
	} else {
		b.send("Checkpoint deleted.")
	}
}

func (b *botAPI) shutdown() {
	b.closed = true
	b.StopReceivingUpdates()
}

// Run starts the bot and returns when the bot has finished processing all receipts.
func Run(conf *config.Config) {
	t0 := time.Now()

	telegramClient, err := tgbotapi.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		logrus.Fatalf("error creating Telegram Bot API client: %v", err)
	}
	logrus.Infof("Authenticated on Telegram bot account %s", telegramClient.Self.UserName)

	checkpointManager, err := checkpoint.NewManager()
	if err != nil {
		logrus.Fatalf("error creating checkpoint manager: %v", err)
	}

	bot := &botAPI{
		BotAPI:  telegramClient,
		Manager: checkpointManager,
		chatID:  conf.Telegram.ChatID,
	}
	defer bot.Close()
	bot.send("Hi, I was started with the nonce '%s'.", conf.Nonce)

	// timeout thread
	shutdownChannel := make(chan struct{})
	go func() {
		defer bot.shutdown()
		timer := time.NewTimer(botTimeoutWatchInterval)
		for {
			select {
			case <-timer.C:
				if botTimeout <= time.Since(t0) {
					bot.send("Timed out waiting for input. Shutting down.")
					return
				}
				timer.Reset(botTimeoutWatchInterval)
			case <-shutdownChannel:
				bot.send("Okay, I will shutdown.")
				if !timer.Stop() {
					<-timer.C
				}
				return
			}
		}
	}()

	// bot state
	botState := botStateIdle
	var receipt models.Receipt
	var payer models.ReceiptItemOwner
	var nextReceiptItem int
	lastModifiedReceiptItem := -1

	if err := checkpointManager.LoadCheckpoint(&receipt); err != nil {
		if !errors.Is(err, checkpoint.ErrCheckpointNotExist) {
			bot.send("I had an unexpected error loading the checkpoint: %v", err)
		}
		receipt = nil
	} else {
		bot.send("Checkpoint found.")
		for nextReceiptItem < receipt.Len() && receipt[nextReceiptItem].Owner != "" {
			nextReceiptItem++
		}
		if nextReceiptItem == receipt.Len() {
			bot.sendPayerChoice(receipt)
			botState = botStateWaitingForPayer
		} else {
			bot.sendReceiptItem(receipt[nextReceiptItem], lastModifiedReceiptItem)
			botState = botStateParsingReceiptInteractively
		}
	}

	resetState := func() {
		botState = botStateIdle
		receipt = nil
		payer = ""
		nextReceiptItem = 0
		lastModifiedReceiptItem = -1
	}

	updatesConfig := tgbotapi.NewUpdate(0 /*offset*/)
	updatesConfig.Timeout = 60
	skippedFirstUpdate := false
	for update := range bot.GetUpdatesChan(updatesConfig) {
		if bot.closed || update.Message == nil || update.Message.Chat.ID != conf.Telegram.ChatID {
			continue
		}
		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)

		if botState != botStateIdle && msg == "/abort" {
			resetState()
			bot.deleteCheckpoint()
			bot.send("Okay, then start a new receipt.")
			continue
		}

		if msg == "/uptime" {
			bot.send("I'm up for %s.", time.Since(t0))
			continue
		}

		if msg == "/finish" {
			close(shutdownChannel)
			continue
		}

		switch botState {
		case botStateIdle:
			receipt = models.ParseReceipt(msg)
			if receipt.Len() > 0 {
				bot.storeCheckpoint(receipt)
				bot.sendReceiptItem(receipt[0], lastModifiedReceiptItem)
				botState = botStateParsingReceiptInteractively
			} else if skippedFirstUpdate {
				bot.send("Could not parse the receipt, try again.")
			} else {
				skippedFirstUpdate = true
			}
		case botStateParsingReceiptInteractively:
			owner := models.ReceiptItemOwner(msg)
			if owner != models.Ana &&
				owner != models.Matheus &&
				owner != models.Shared &&
				owner != models.NotReceiptItem &&
				owner != delayDecision &&
				(owner != backOneItem || lastModifiedReceiptItem < 0) {
				bot.sendOwnerChoice(lastModifiedReceiptItem)
			} else {
				// process owner input
				if owner != delayDecision && owner != backOneItem {
					receipt[nextReceiptItem].Owner = owner
					lastModifiedReceiptItem = nextReceiptItem
					nextReceiptItem = (nextReceiptItem + 1) % receipt.Len()
					for receipt[nextReceiptItem].Owner != "" && nextReceiptItem != lastModifiedReceiptItem {
						nextReceiptItem = (nextReceiptItem + 1) % receipt.Len()
					}
					bot.storeCheckpoint(receipt)
				} else if owner == delayDecision {
					nextReceiptItem = (nextReceiptItem + 1) % receipt.Len()
					for receipt[nextReceiptItem].Owner != "" {
						nextReceiptItem = (nextReceiptItem + 1) % receipt.Len()
					}
				} else { // owner == backOneItem
					nextReceiptItem = lastModifiedReceiptItem
					lastModifiedReceiptItem = -1
					receipt[nextReceiptItem].Owner = ""
					bot.storeCheckpoint(receipt)
				}

				if receipt[nextReceiptItem].Owner == "" {
					bot.sendReceiptItem(receipt[nextReceiptItem], lastModifiedReceiptItem)
				} else {
					bot.sendPayerChoice(receipt)
					botState = botStateWaitingForPayer
				}
			}
		case botStateWaitingForPayer:
			payer = models.ReceiptItemOwner(msg)
			if payer != models.Ana && payer != models.Matheus {
				bot.send("Invalid choice. Choose one of {%s, %s}.", models.Ana, models.Matheus)
			} else {
				bot.send("Please type in the name of the store.")
				botState = botStateWaitingForStore
			}
		case botStateWaitingForStore:
			store := msg
			if len(store) == 0 {
				bot.send("Store name cannot be empty.")
			} else {
				nonSharedExpense, sharedExpense := receipt.ComputeExpenses(payer)

				bot.send("Creating non-shared expense...")
				expenseMsg := splitwise.CreateExpense(&conf.Splitwise, store, nonSharedExpense)
				bot.send(expenseMsg)

				bot.send("Creating shared expense...")
				expenseMsg = splitwise.CreateExpense(&conf.Splitwise, store, sharedExpense)
				bot.send(expenseMsg)

				bot.deleteCheckpoint()
				bot.send("More receipts?")
				resetState()
			}
		default:
			logrus.Errorf("invalid botState: %d", botState)
		}
	}

	bot.send("Cya.")
}
