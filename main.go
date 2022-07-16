package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type (
	botAPI struct {
		*tgbotapi.BotAPI

		chatID int64
	}

	botState         int
	receiptItemOwner string

	userShare struct {
		userID int64
		paid   priceInCents
		owed   priceInCents
	}
)

const (
	botStateIdle botState = iota
	botStateParsingReceiptInteractively
	botStateWaitingForPayer

	ana            receiptItemOwner = "a"
	matheus        receiptItemOwner = "m"
	shared         receiptItemOwner = "s"
	delay          receiptItemOwner = "d"
	notReceiptItem receiptItemOwner = "n"

	zeroCents = priceInCents(0)
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
	var receiptItems []*receiptItem
	totalInCents := make(map[receiptItemOwner]priceInCents)

	resetState := func() {
		botState = botStateIdle
		receiptItems = nil
		totalInCents = make(map[receiptItemOwner]priceInCents)
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
			item.mainLine,
			item.priceInCents.Format(),
			item.nextLine,
			ana,
			matheus,
			shared,
			delay,
			notReceiptItem,
		)
	}

	updatesConfig := tgbotapi.NewUpdate(0 /*offset*/)
	updatesConfig.Timeout = 60
	for update := range bot.GetUpdatesChan(updatesConfig) {
		if update.Message == nil || update.Message.Chat.ID != chatID {
			continue
		}
		msg := update.Message.Text
		logrus.Infof("[%s] %s", update.Message.From.UserName, msg)

		if botState != botStateIdle && msg == "/abort" {
			resetState()
			bot.Send("Cya.")
			continue
		}

		switch botState {
		case botStateIdle:
			receiptItems = parseReceipt(msg)
			if len(receiptItems) > 0 {
				printNextReceiptItem()
				botState = botStateParsingReceiptInteractively
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
					totalInCents[owner] += item.priceInCents
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
			payer := receiptItemOwner(msg)
			if payer != ana && payer != matheus {
				bot.Send("Invalid choice. Choose one of {%s, %s}.", ana, matheus)
			} else {
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

func createExpense(bot *botAPI, groupID int64, description string, cost priceInCents, user0, user1 *userShare) {
	if cost < 0 {
		bot.Send("Skipping expense with negative cost.")
		return
	}

	if cost == 0 {
		bot.Send("Skipping expense with cost zero.")
		return
	}

	var expense bytes.Buffer
	err := json.NewEncoder(&expense).Encode(map[string]interface{}{
		"currency_code":        "EUR",
		"category_id":          12, // Groceries
		"description":          fmt.Sprintf("groceries %s", description),
		"cost":                 cost.Format(),
		"group_id":             groupID,
		"users__0__user_id":    user0.userID,
		"users__0__paid_share": user0.paid.Format(),
		"users__0__owed_share": user0.owed.Format(),
		"users__1__user_id":    user1.userID,
		"users__1__paid_share": user1.paid.Format(),
		"users__1__owed_share": user1.owed.Format(),
	})
	if err != nil {
		bot.Send("Error encoding Splitwise JSON body: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, "https://secure.splitwise.com/api/v3.0/create_expense", &expense)
	if err != nil {
		bot.Send("Error creating request for Splitwise API: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("SPLITWISE_TOKEN")))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		bot.Send("Error POSTing expense to Splitwise API: %v", err)
		return
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		bot.Send("Splitwise API call returned %d, but an error occurred reading the payload: %v", resp.StatusCode, err)
	} else if !strings.Contains(string(b), `"expenses":[{`) {
		bot.Send("Splitwise API call returned %d: %s", resp.StatusCode, string(b))
	} else {
		bot.Send("Expense successfully created on the Splitwise API.")
	}
}
