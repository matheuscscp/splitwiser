package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
		user           models.ReceiptItemOwner
		closed         bool
		msgQueue       []string
		updateChannel  tgbotapi.UpdatesChannel

		// state
		chatMode bool
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

	notReceiptItem   = "n"
	resetReceipt     = "r"
	newPrice         = "p"
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
	b.send(`%s (%s)

Please choose the owner:
%s - Set owned by Ana
%s - Set owned by Matheus
%s - Set owned by both (shared)
%s - Not a receipt item
%s - Reset receipt
%s <new_price> - Set new price
%s - Delay item decision%s`,
		item.Name,
		item.Price,
		models.Ana,
		models.Matheus,
		models.Shared,
		notReceiptItem,
		resetReceipt,
		newPrice,
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
		"Invalid choice. Choose one of {%s, %s, %s, %s, %s, %s, %s%s}.",
		models.Ana, models.Matheus, models.Shared, notReceiptItem, resetReceipt, newPrice, delayDecision, undo,
	)
}

func (b *botClient) sendPayerChoice(receipt models.Receipt) {
	ownerTotals, total, totalWithDiscounts := receipt.ComputeTotals()
	b.send(`Ana's total: %v
Matheus' total: %v
Shared total: %v
Total: %v
Total with discounts: %v

Please choose the payer:
%s - Ana
%s - Matheus
%s - Reset receipt`,
		ownerTotals[models.Ana],
		ownerTotals[models.Matheus],
		ownerTotals[models.Shared],
		total,
		totalWithDiscounts,
		models.Ana,
		models.Matheus,
		resetReceipt,
	)
}

func (b *botClient) sendMoreReceipts() {
	b.send("More receipts?")
}

func (bc *botClient) handlePhoto(ctx context.Context, message *tgbotapi.Message) models.Receipt {
	bc.send("M'kay, I'm sending this image to OpenAI for processing...")
	fd, err := bc.telegramClient.GetFile(tgbotapi.FileConfig{
		FileID: message.Photo[len(message.Photo)-1].FileID,
	})
	if err != nil {
		bc.send("I got this error trying to get a descriptor for the file you sent me:\n\n%v", err)
		return nil
	}
	f, err := http.Get(fd.Link(bc.conf.Telegram.Token))
	if err != nil {
		bc.send("I got this error trying to get the file you sent me:\n\n%v", err)
		return nil
	}
	defer f.Body.Close()
	b, err := io.ReadAll(f.Body)
	if err != nil {
		bc.send("I got this error trying to download the file you sent me:\n\n%v", err)
		return nil
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

I need you to parse the photo and return the items in the exact example JSON format below, because I'm
not as smart as you and I need the items to be in this simple text format so my Go code can understand
it easily.

Please output only the items like in the example format below, and nothing else. Please don't write
any greeting messages or anything like that, because that makes it harder for me to parse your
results. Just return me a valid JSON array like the one below. Please do not include the backticks
wrapper.

If there are fees at the end of the receipt photo, please include these fees as items.
Discounts should also be included and have negative prices.

Finally, here goes the example JSON format:

[
	{"name":"Smoky BBQ wings","euro_cents":399},
	{"name":"Smoky BBQ wings Discount","euro_cents":-399},
	{"name":"PopChips BBQ 5pk","euro_cents":249},
	{"name":"RedHen Chicken Dippe","euro_cents":155},
	{"name":"Whole Milk 2L","euro_cents":209},
	{"name":"Coca Cola Regular","euro_cents":620},
	{"name":"Ready Salted Crisps","euro_cents":119},
	{"name":"Hummus Chips","euro_cents":149},
	{"name":"Vegan Ice Sticks Alm","euro_cents":299}
]`,
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
	var lastMessageFromOpenAI openai.ChatCompletionMessage
	var receipt models.Receipt
	parsePhoto := func() error {
		for i := 0; i < 3; i++ {
			resp, err := bc.openAI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				MaxTokens: 4096,
				Model:     openai.GPT4o,
				Messages:  messages,
			})
			if err != nil {
				bc.send("OpenAI replied an error:\n\n%v", err)
				return err
			}
			lastMessageFromOpenAI = resp.Choices[0].Message
			if err := json.Unmarshal([]byte(lastMessageFromOpenAI.Content), &receipt); err != nil {
				bc.send(`OpenAI replied an invalid JSON. This is a dumb error, I'm just gonna retry for you.

Error: %v

Content:

%s`, err, lastMessageFromOpenAI.Content)
				continue
			}
			return nil
		}
		return errors.New("OpenAI replied an invalid JSON 3 times in a row, I'm giving up.")
	}
	askIfResultIsEnough := func() {
		bc.send(`Here are the items and prices from OpenAI:

%s

Check if OpenAI forgot any items that are part of the receipt. Fees and discounts should be included.

To continue parsing this receipt, enter y/yes.

To abort this receipt, enter n/no.

To send a follow-up message to OpenAI asking for changes in this receipt, just type in a prompt in natural language.`,
			receipt,
		)
	}
	if parsePhoto() != nil {
		return nil
	}
	askIfResultIsEnough()

	for {
		select {
		case <-ctx.Done():
			return nil
		case followup := <-bc.updateChannel:
			if followup.Message == nil {
				continue
			}
			if bc.isToggleChat(followup.Message) {
				bc.handleToggleChat()
				continue
			}
			if bc.shouldSkip(followup.Message) {
				continue
			}
			msg := followup.Message.Text
			switch tl := strings.ToLower(strings.TrimSpace(msg)); {
			case tl == "y" || tl == "yes":
				return receipt
			case tl == "n" || tl == "no":
				bc.sendMoreReceipts()
				return nil
			}
			bc.send("M'kay, I'm forwarding this follow-up prompt to OpenAI...")
			messages = append(messages, lastMessageFromOpenAI, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: strings.TrimSpace(msg),
			})
			if parsePhoto() != nil {
				return nil
			}
			askIfResultIsEnough()
		}
	}
}

func (b *botClient) shouldSkip(message *tgbotapi.Message) bool {
	return b.closed ||
		b.chatMode ||
		message.Chat.ID != b.chatID ||
		userFromMessage(message) != b.user ||
		regexCya.MatchString(message.Text)
}

func userFromMessage(message *tgbotapi.Message) models.ReceiptItemOwner {
	if message.From.UserName == "matheuscscp" {
		return models.Matheus
	}
	return models.Ana
}

func (b *botClient) isToggleChat(message *tgbotapi.Message) bool {
	return message.Chat.ID == b.chatID && message.Text == "/togglechat"
}

func (b *botClient) handleToggleChat() {
	b.chatMode = !b.chatMode
	state := "enabled"
	if !b.chatMode {
		state = "disabled"
	}
	b.send("Chat mode is now %s.", state)
}

// Run starts the bot and returns when the bot has finished processing all receipts.
func Run(ctx context.Context, user models.ReceiptItemOwner) error {
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
		user:           user,
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
	bot.enqueue("Hi, %s.", user.Pretty())
	if err := checkpointService.Load(ctx, &receipt); err != nil {
		if !errors.Is(err, checkpoint.ErrCheckpointNotExist) {
			bot.enqueue("I had an unexpected error loading the checkpoint: %v", err)
		}
		bot.send("Let's parse a receipt. Please send it my way. I can understand screenshots, photographs and text messages.")
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

	softResetState := func() {
		payer = ""
		nextReceiptItem = 0
		lastModifiedReceiptItem = -1
	}

	resetState := func() {
		if err := checkpointService.Delete(ctx); err != nil {
			bot.enqueue("I had an unexpected error deleting the checkpoint: %v", err)
		} else {
			bot.enqueue("Checkpoint deleted.")
		}

		botState = botStateIdle
		receipt = nil
		softResetState()

		bot.sendMoreReceipts()
	}

	softResetOption := func() {
		bot.send("M'kay, let's go back to the beginning of this receipt:\n\n%s", receipt)
		softResetState()
		for _, item := range receipt {
			item.Owner = ""
		}
		storeCheckpoint()
		botState = botStateParsingReceiptInteractively
		bot.sendReceiptItem(receipt[0], lastModifiedReceiptItem)
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
		if update.Message == nil {
			continue
		}

		message := update.Message
		if bot.isToggleChat(message) {
			bot.handleToggleChat()
			continue
		}

		if bot.shouldSkip(message) {
			continue
		}

		logrus.Infof("[%s] %s", message.From.UserName, message.Text)
		logrus.WithField("msg", message).Debug("msg")

		// handle commands
		if botState != botStateIdle && message.Text == "/abort" {
			resetState()
			continue
		}
		if message.Text == "/uptime" {
			bot.send("I'm up for %s.", time.Since(startTime))
			continue
		}
		if message.Text == "/finish" {
			cancel()
			continue
		}

		switch botState {
		case botStateIdle:
			if len(message.Photo) > 0 {
				receipt = bot.handlePhoto(ctx, message)
			} else {
				receipt = models.ParseReceipt(message.Text)
				if receipt.Len() == 0 {
					bot.send("I can't understand that. Let's try again.")
				} else {
					bot.send("Let's parse the following receipt:\n\n%s", receipt)
				}
			}
			if receipt.Len() > 0 {
				storeCheckpoint()
				bot.sendReceiptItem(receipt[0], lastModifiedReceiptItem)
				botState = botStateParsingReceiptInteractively
			}
		case botStateParsingReceiptInteractively:
			message.Text = strings.TrimSpace(strings.ToLower(message.Text))
			switch {
			case message.Text == string(models.Ana) || message.Text == string(models.Matheus) || message.Text == string(models.Shared) || message.Text == notReceiptItem:
				receipt[nextReceiptItem].Owner = models.ReceiptItemOwner(message.Text)
				lastModifiedReceiptItem = nextReceiptItem
				nextReceiptItem = receipt.NextItem(nextReceiptItem)
				for receipt[nextReceiptItem].Owner != "" && nextReceiptItem != lastModifiedReceiptItem {
					nextReceiptItem = receipt.NextItem(nextReceiptItem)
				}
				storeCheckpoint()
			case message.Text == resetReceipt:
				softResetOption()
				continue
			case strings.HasPrefix(message.Text, newPrice+" "):
				price, ok := models.ParsePriceInCents(message.Text[len(newPrice+" "):])
				if !ok {
					bot.send("I can't understand that price, please try again.")
					continue
				}
				receipt[nextReceiptItem].Price = price
				storeCheckpoint()
			case message.Text == delayDecision:
				nextReceiptItem = receipt.NextItem(nextReceiptItem)
				for receipt[nextReceiptItem].Owner != "" {
					nextReceiptItem = receipt.NextItem(nextReceiptItem)
				}
			case message.Text == undoLastDecision && lastModifiedReceiptItem >= 0:
				nextReceiptItem = lastModifiedReceiptItem
				lastModifiedReceiptItem = -1
				receipt[nextReceiptItem].Owner = ""
				storeCheckpoint()
			default:
				bot.sendOwnerChoice(lastModifiedReceiptItem)
				continue
			}

			if receipt[nextReceiptItem].Owner == "" {
				bot.sendReceiptItem(receipt[nextReceiptItem], lastModifiedReceiptItem)
			} else {
				bot.sendPayerChoice(receipt)
				botState = botStateWaitingForPayer
			}
		case botStateWaitingForPayer:
			payer = models.ReceiptItemOwner(strings.TrimSpace(strings.ToLower(message.Text)))
			if payer != models.Ana && payer != models.Matheus && payer != resetReceipt {
				bot.send("Invalid choice. Choose one of {%s, %s, %s}.", models.Ana, models.Matheus, resetReceipt)
			} else if payer == resetReceipt {
				softResetOption()
			} else {
				bot.send("Please type in the name of the store.")
				botState = botStateWaitingForStore
			}
		case botStateWaitingForStore:
			storeName := message.Text
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
