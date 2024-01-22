package bot

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/matheuscscp/splitwiser/config"
	_ "github.com/matheuscscp/splitwiser/logging"
	"github.com/matheuscscp/splitwiser/models"
	"github.com/matheuscscp/splitwiser/pkg/splitwise"
	"github.com/matheuscscp/splitwiser/services/checkpoint"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type (
	botClient struct {
		conf           *config.Bot
		openAI         *openai.Client
		telegramClient *tgbotapi.BotAPI
		chatID         int64
		closed         bool
		msgQueue       []string
		updateChannel  tgbotapi.UpdatesChannel
	}

	botState int
)

const (
	botStateIdle botState = iota
	botStateParsingReceiptInteractively
	botStateWaitingForPayer
	botStateWaitingForStore

	botLongPollingTimeout = 60 * time.Second
	botTimeout            = 540*time.Second - botLongPollingTimeout - 5*time.Second

	delayDecision    = "d"
	undoLastDecision = "u"
)

var (
	regexCya = regexp.MustCompile(`(?i)^\s*cya\s*$`)
)

func (b *botClient) account() string {
	return b.telegramClient.Self.UserName
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
	b.send(`Item: "%s" (%v)

(see if the next line rings a bell: "%s")

Please choose the owner:
%s - Ana
%s - Matheus
%s - Shared
%s - Not a receipt item
%s - Delay item decision%s`,
		item.MainLine,
		item.Price,
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
	totalsInCents, totalInCents := receipt.ComputeTotals()
	b.send(`Ana's total: %v
Matheus's total: %v
Shared total: %v
Overall total: %v

Please choose the payer:
%s - Ana
%s - Matheus`,
		totalsInCents[models.Ana],
		totalsInCents[models.Matheus],
		totalsInCents[models.Shared],
		totalInCents,
		models.Ana,
		models.Matheus,
	)
}

func (bc *botClient) handlePhoto(ctx context.Context, update *tgbotapi.Update) string {
	bc.send("Okay, I'm sending this image to OpenAI for processing...")
	fd, err := bc.telegramClient.GetFile(tgbotapi.FileConfig{
		FileID: update.Message.Photo[len(update.Message.Photo)-1].FileID,
	})
	if err != nil {
		bc.send("I got this error trying to get a descriptor for the file you sent me:\n\n%v", err)
		return ""
	}
	f, err := http.Get(fd.Link(bc.conf.Telegram.Token))
	if err != nil {
		bc.send("I got this error trying to get the file you sent me:\n\n%v", err)
		return ""
	}
	defer f.Body.Close()
	b, err := io.ReadAll(f.Body)
	if err != nil {
		bc.send("I got this error trying to download the file you sent me:\n\n%v", err)
		return ""
	}
	image := base64.StdEncoding.EncodeToString(b)
	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: `Hi! I'm Matheus' Telegram Bot for parsing his domestic receipts.

Matheus programmed me to ask for your help when he sends photographs of his receipts to me.

Please find attached a base64-encoded photograph of a receipt that Matheus sent to me.

I need you to parse the photo and return the items in the exact example format below, because I'm
not as smart as you and I need the items to be in this simple text format so my Go code can understand
it easily.

Please output only the items like in the example format below, and nothing else. Please don't write
any greeting messages or anything like that, because that makes it harder for me to parse your
results.

Please send the result in a single long line, like the example below!

If there are fees at the end of the receipt photo, please include these fees as items.

Finally, here goes the example format:

Smoky BBQ wings 3.99 A PopChips BBQ 5pk 2.49 C RedHen Chicken Dippe 1.55 A Whole Milk 2L 2.09 A Coca Cola Regular 6.20 C 2 x 3.10 Ready Salted Crisps 1.19 C Hummus Chips 1.49 C Vegan Ice Sticks Alm 2.99 C ------------ TOTAL 21.99
`,
				},
				{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL: fmt.Sprintf("data:image/jpeg;base64,%s", image),
					},
				},
			},
		},
	}
	resp, err := bc.openAI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		MaxTokens: 4096,
		Model:     openai.GPT4VisionPreview,
		Messages:  messages,
	})
	if err != nil {
		bc.send("I got this error trying to upload the file you sent me to OpenAI:\n\n%v", err)
		return ""
	}
	content := resp.Choices[0].Message.Content
	askIfResultIsEnough := func() {
		bc.send("This is what OpenAI sent me:\n\n%s\n\nShould I parse this? (Yes/No or a follow-up message for OpenAI)", content)
	}
	askIfResultIsEnough()

	for {
		select {
		case <-ctx.Done():
			return ""
		case followup := <-bc.updateChannel:
			if bc.shouldSkip(&followup) {
				continue
			}
			msg := followup.Message.Text
			switch tl := strings.ToLower(msg); {
			case tl == "y" || tl == "yes":
				return content
			case tl == "n" || tl == "no":
				return ""
			}
			bc.send("Okay I'm forwarding this follow-up prompt to OpenAI...")
			messages = append(messages, resp.Choices[0].Message, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: strings.TrimSpace(msg),
			})
			resp, err := bc.openAI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				MaxTokens: 4096,
				Model:     openai.GPT4VisionPreview,
				Messages:  messages,
			})
			if err != nil {
				bc.send("I got this error following up with OpenAI:\n\n%v", err)
				return ""
			}
			content = resp.Choices[0].Message.Content
			askIfResultIsEnough()
		}
	}
}

func (b *botClient) shouldSkip(update *tgbotapi.Update) bool {
	if update.Message.Chat.ID != b.conf.Telegram.ChatID {
		logrus.WithField("chat_id", update.Message.Chat.ID).Warn("unallowed chat id")
	}
	return b.closed ||
		update.Message == nil ||
		update.Message.Chat.ID != b.conf.Telegram.ChatID ||
		regexCya.MatchString(update.Message.Text)
}

// Run starts the bot and returns when the bot has finished processing all receipts.
func Run(ctx context.Context) error {
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(ctx, botTimeout)
	defer cancel()

	var conf config.Bot
	if err := config.Load(&conf); err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	openAI := openai.NewClient(conf.OpenAI.Token)

	telegramClient, err := tgbotapi.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		return fmt.Errorf("error creating Telegram Bot API client: %w", err)
	}

	splitwiseClient := splitwise.NewClient(&conf.Splitwise)

	checkpointService, err := checkpoint.NewService(ctx, conf.CheckpointBucket)
	if err != nil {
		return fmt.Errorf("error creating checkpoint service: %w", err)
	}
	defer checkpointService.Close()

	updateConf := tgbotapi.NewUpdate(0 /*offset*/)
	updateConf.Timeout = int(botLongPollingTimeout.Seconds())
	updateChannel := telegramClient.GetUpdatesChan(updateConf)

	bot := &botClient{
		conf:           &conf,
		openAI:         openAI,
		telegramClient: telegramClient,
		chatID:         conf.Telegram.ChatID,
		updateChannel:  updateChannel,
	}

	logrus.Infof("Authenticated on Telegram bot account %s", bot.account())

	// shutdown thread
	go func() {
		<-ctx.Done()
		bot.send("My context was cancelled, I'm shutting down.")
		bot.shutdown()
	}()

	// bot state
	botState := botStateIdle
	var receipt models.Receipt
	var payer models.ReceiptItemOwner
	var nextReceiptItem int
	lastModifiedReceiptItem := -1

	// load checkpoint
	bot.enqueue("Hi.")
	if err := checkpointService.Load(ctx, &receipt); err != nil {
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
		if err := checkpointService.Store(ctx, receipt); err != nil {
			bot.enqueue("I had an unexpected error storing the checkpoint: %v", err)
		}
	}

	resetState := func() {
		if err := checkpointService.Delete(ctx); err != nil {
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
		msg := splitwiseClient.CreateExpense(ctx, expense, storeName)
		bot.enqueue(msg)
	}
	createNonSharedExpense := func(expense *models.Expense, storeName string) {
		createExpense("non-shared", expense, storeName)
	}
	createSharedExpense := func(expense *models.Expense, storeName string) {
		createExpense("shared", expense, storeName)
	}

	for update := range updateChannel {
		if bot.shouldSkip(&update) {
			continue
		}

		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)
		logrus.WithField("msg", update.Message).Debug("msg")

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
			cancel()
			continue
		}

		switch botState {
		case botStateIdle:
			// handle photo input
			if len(update.Message.Photo) > 0 {
				msg = bot.handlePhoto(ctx, &update)
				if msg == "" {
					continue
				}
			}

			func() {
				defer func() {
					if p := recover(); p != nil {
						receipt = nil
						logrus.Errorf("panic parsing input as receipt: %v\n%s", p, string(debug.Stack()))
					}
				}()
				receipt = models.ParseReceipt(msg)
			}()

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
