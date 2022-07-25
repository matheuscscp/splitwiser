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

func (b *botAPI) getUpdatesChannel() tgbotapi.UpdatesChannel {
	conf := tgbotapi.NewUpdate(0 /*offset*/)
	conf.Timeout = 60
	return b.GetUpdatesChan(conf)
}

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

func (b *botAPI) shutdown() {
	b.closed = true
	b.StopReceivingUpdates()
}

// Run starts the bot and returns when the bot has finished processing all receipts.
func Run(nonce string) error {
	startTime := time.Now()

	conf, err := config.Load()
	if err != nil {
		return err
	}

	telegramClient, err := tgbotapi.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		return fmt.Errorf("error creating Telegram Bot API client: %w", err)
	}

	checkpointManager, err := checkpoint.NewManager(conf.CheckpointBucket)
	if err != nil {
		return fmt.Errorf("error creating checkpoint manager: %w", err)
	}

	bot := &botAPI{
		BotAPI:  telegramClient,
		Manager: checkpointManager,
		chatID:  conf.Telegram.ChatID,
	}
	defer bot.Close()

	logrus.Infof("Authenticated on Telegram bot account %s", telegramClient.Self.UserName)
	bot.send("Hi, I was started with the nonce '%s'.", nonce)

	// timeout thread
	shutdownChannel := make(chan struct{})
	go func() {
		defer bot.shutdown()
		timer := time.NewTimer(botTimeoutWatchInterval)
		for {
			select {
			case <-timer.C:
				if botTimeout <= time.Since(startTime) {
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

	// load checkpoint
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
		if err := checkpointManager.DeleteCheckpoint(); err != nil {
			bot.send("I had an unexpected error deleting the checkpoint: %v", err)
		} else {
			bot.send("Checkpoint deleted.")
		}

		botState = botStateIdle
		receipt = nil
		payer = ""
		nextReceiptItem = 0
		lastModifiedReceiptItem = -1

		bot.send("More receipts?")
	}

	for update := range bot.getUpdatesChannel() {
		if bot.closed ||
			update.Message == nil ||
			update.Message.Chat.ID != conf.Telegram.ChatID ||
			update.Message.Text == "cya" {
			continue
		}
		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)

		if botState != botStateIdle && msg == "/abort" {
			resetState()
			continue
		}

		if msg == "/uptime" {
			bot.send("I'm up for %s.", time.Since(startTime))
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
			} else {
				bot.send("Could not parse the receipt, try again.")
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
					nextReceiptItem = receipt.NextItem(nextReceiptItem)
					for receipt[nextReceiptItem].Owner != "" && nextReceiptItem != lastModifiedReceiptItem {
						nextReceiptItem = receipt.NextItem(nextReceiptItem)
					}
					bot.storeCheckpoint(receipt)
				} else if owner == delayDecision {
					nextReceiptItem = receipt.NextItem(nextReceiptItem)
					for receipt[nextReceiptItem].Owner != "" {
						nextReceiptItem = receipt.NextItem(nextReceiptItem)
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

				resetState()
			}
		default:
			bot.send("My state machine led me to an invalid state: %v.", botState)
		}
	}

	bot.send("Cya.")
	return nil
}
