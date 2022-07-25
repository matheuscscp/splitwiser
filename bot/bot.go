package bot

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
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
	botClient struct {
		telegramClient *tgbotapi.BotAPI
		chatID         int64
		closed         bool
		msgQueue       []string
	}

	botState int
)

const (
	botStateIdle botState = iota
	botStateParsingReceiptInteractively
	botStateWaitingForPayer
	botStateWaitingForStore

	botTimeout              = 510 * time.Second
	botTimeoutWatchInterval = 5 * time.Second

	delayDecision    = "d"
	undoLastDecision = "u"
)

var (
	regexCya = regexp.MustCompile(`(?i)^\s*cya\s*$`)
)

func (b *botClient) account() string {
	return b.telegramClient.Self.UserName
}

func (b *botClient) getUpdatesChannel() tgbotapi.UpdatesChannel {
	conf := tgbotapi.NewUpdate(0 /*offset*/)
	conf.Timeout = 60
	return b.telegramClient.GetUpdatesChan(conf)
}

func (b *botClient) shutdown() {
	b.closed = true
	b.telegramClient.StopReceivingUpdates()
}

func (b *botClient) enqueue(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	b.msgQueue = append(b.msgQueue, msg)
}

func (b *botClient) send(format string, args ...interface{}) {
	b.enqueue(format, args...)

	fullText := strings.Join(b.msgQueue, "\n\n")
	logrus.Infof("[%s] %s", b.account(), fullText)

	msg := tgbotapi.NewMessage(b.chatID, fullText)
	if _, err := b.telegramClient.Send(msg); err != nil {
		logrus.Errorf("error sending message: %v\n\nmessage text:\n%s", err, fullText)
	} else {
		b.msgQueue = nil
	}
}

func (b *botClient) sendReceiptItem(item *models.ReceiptItem, lastModifiedReceiptItem int) {
	var undo string
	if lastModifiedReceiptItem >= 0 {
		undo = fmt.Sprintf("\n%s - Undo last decision", undoLastDecision)
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
		undo,
	)
}

func (b *botClient) sendOwnerChoice(lastModifiedReceiptItem int) {
	var undo string
	if lastModifiedReceiptItem >= 0 {
		undo = fmt.Sprintf(", %s", undoLastDecision)
	}
	b.send(
		"Invalid choice. Choose one of {%s, %s, %s, %s, %s%s}.",
		models.Ana, models.Matheus, models.Shared, models.NotReceiptItem, delayDecision, undo,
	)
}

func (b *botClient) sendPayerChoice(receipt models.Receipt) {
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

// Run starts the bot and returns when the bot has finished processing all receipts.
func Run() error {
	startTime := time.Now()

	conf, err := config.Load()
	if err != nil {
		return err
	}

	telegramClient, err := tgbotapi.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		return fmt.Errorf("error creating Telegram Bot API client: %w", err)
	}

	splitwiseClient := splitwise.NewClient(&conf.Splitwise)

	checkpointService, err := checkpoint.NewService(conf.CheckpointBucket)
	if err != nil {
		return fmt.Errorf("error creating checkpoint service: %w", err)
	}
	defer checkpointService.Close()

	bot := &botClient{
		telegramClient: telegramClient,
		chatID:         conf.Telegram.ChatID,
	}

	logrus.Infof("Authenticated on Telegram bot account %s", bot.account())

	// shutdown thread
	shutdownThread := make(chan struct{})
	go func() {
		defer bot.shutdown()
		timer := time.NewTimer(botTimeoutWatchInterval)
		for {
			select {
			case <-timer.C:
				if botTimeout <= time.Since(startTime) {
					bot.send("You took too long to respond, I'm shutting down.")
					return
				}
				timer.Reset(botTimeoutWatchInterval)
			case <-shutdownThread:
				bot.send("Okay, I'm shutting down.")
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
	bot.enqueue("Hi.")
	if err := checkpointService.Load(&receipt); err != nil {
		if !errors.Is(err, checkpoint.ErrCheckpointNotExist) {
			bot.enqueue("I had an unexpected error loading the checkpoint: %v", err)
		}
		bot.send("Let's start a new receipt. Please send it my way.")
		receipt = nil
	} else {
		bot.enqueue("I found a previous receipt, let's finish it.")
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

	storeCheckpoint := func() {
		if err := checkpointService.Store(receipt); err != nil {
			bot.enqueue("I had an unexpected error storing the checkpoint: %v", err)
		}
	}

	resetState := func() {
		if err := checkpointService.Delete(); err != nil {
			bot.enqueue("I had an unexpected error deleting the checkpoint: %v", err)
		} else {
			bot.enqueue("Checkpoint deleted.")
		}

		botState = botStateIdle
		receipt = nil
		payer = ""
		nextReceiptItem = 0
		lastModifiedReceiptItem = -1

		bot.send("More receipts?")
	}

	createExpense := func(expenseType string, expense *models.Expense, storeName string) {
		bot.send("Creating %s expense...", expenseType)
		msg := splitwiseClient.CreateExpense(expense, storeName)
		bot.enqueue(msg)
	}
	createNonSharedExpense := func(expense *models.Expense, storeName string) {
		createExpense("non-shared", expense, storeName)
	}
	createSharedExpense := func(expense *models.Expense, storeName string) {
		createExpense("shared", expense, storeName)
	}

	for update := range bot.getUpdatesChannel() {
		if bot.closed ||
			update.Message == nil ||
			update.Message.Chat.ID != conf.Telegram.ChatID ||
			regexCya.MatchString(update.Message.Text) {
			continue
		}
		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)

		// handle commands
		if botState != botStateIdle && msg == "/abort" {
			resetState()
			continue
		}
		if msg == "/uptime" {
			bot.send("I'm up for %s.", time.Since(startTime))
			continue
		}
		if msg == "/finish" {
			close(shutdownThread)
			continue
		}

		switch botState {
		case botStateIdle:
			receipt = models.ParseReceipt(msg)
			if receipt.Len() > 0 {
				storeCheckpoint()
				bot.sendReceiptItem(receipt[0], lastModifiedReceiptItem)
				botState = botStateParsingReceiptInteractively
			} else {
				bot.send("I can't understand that. Let's try again.")
			}
		case botStateParsingReceiptInteractively:
			input := models.ReceiptItemOwner(msg)
			if input != models.Ana &&
				input != models.Matheus &&
				input != models.Shared &&
				input != models.NotReceiptItem &&
				input != delayDecision &&
				(input != undoLastDecision || lastModifiedReceiptItem < 0) {
				bot.sendOwnerChoice(lastModifiedReceiptItem)
			} else {
				// process owner input
				if input != delayDecision && input != undoLastDecision {
					receipt[nextReceiptItem].Owner = input
					lastModifiedReceiptItem = nextReceiptItem
					nextReceiptItem = receipt.NextItem(nextReceiptItem)
					for receipt[nextReceiptItem].Owner != "" && nextReceiptItem != lastModifiedReceiptItem {
						nextReceiptItem = receipt.NextItem(nextReceiptItem)
					}
					storeCheckpoint()
				} else if input == delayDecision {
					nextReceiptItem = receipt.NextItem(nextReceiptItem)
					for receipt[nextReceiptItem].Owner != "" {
						nextReceiptItem = receipt.NextItem(nextReceiptItem)
					}
				} else { // input == undoLastDecision
					nextReceiptItem = lastModifiedReceiptItem
					lastModifiedReceiptItem = -1
					receipt[nextReceiptItem].Owner = ""
					storeCheckpoint()
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
			storeName := msg
			if len(storeName) == 0 {
				bot.send("Store name cannot be empty.")
			} else {
				nonSharedExpense, sharedExpense := receipt.ComputeExpenses(payer)
				createNonSharedExpense(nonSharedExpense, storeName)
				createSharedExpense(sharedExpense, storeName)
				resetState()
			}
		default:
			bot.send("My state machine led me to an invalid state: %v.", botState)
		}
	}

	bot.send("Cya.")
	return nil
}
