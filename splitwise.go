package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/matheuscscp/splitwiser/models"
)

type (
	userShare struct {
		userID int64
		paid   models.PriceInCents
		owed   models.PriceInCents
	}
)

func createExpense(
	bot *botAPI,
	groupID int64,
	store, description string,
	cost models.PriceInCents,
	user0, user1 *userShare,
) {
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
		"description":          fmt.Sprintf("%s %s", store, description),
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
