package splitwise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/matheuscscp/splitwiser/models"
)

type (
	// UserShare ...
	UserShare struct {
		UserID int64
		Paid   models.PriceInCents
		Owed   models.PriceInCents
	}
)

// CreateExpense creates an expense on the Splitwise API and
// returns a message for the bot to send back as the result.
func CreateExpense(
	apiToken string,
	groupID int64,
	store, description string,
	cost models.PriceInCents,
	user0, user1 *UserShare,
) string {
	if cost < 0 {
		return "Skipping expense with negative cost."
	}

	if cost == 0 {
		return "Skipping expense with cost zero."
	}

	// create payload
	var expense bytes.Buffer
	err := json.NewEncoder(&expense).Encode(map[string]interface{}{
		"currency_code":        "EUR",
		"category_id":          12, // Groceries
		"description":          fmt.Sprintf("%s %s", store, description),
		"cost":                 cost.Format(),
		"group_id":             groupID,
		"users__0__user_id":    user0.UserID,
		"users__0__paid_share": user0.Paid.Format(),
		"users__0__owed_share": user0.Owed.Format(),
		"users__1__user_id":    user1.UserID,
		"users__1__paid_share": user1.Paid.Format(),
		"users__1__owed_share": user1.Owed.Format(),
	})
	if err != nil {
		return fmt.Sprintf("Error encoding Splitwise JSON body: %v", err)
	}

	// make request
	req, err := http.NewRequest(http.MethodPost, "https://secure.splitwise.com/api/v3.0/create_expense", &expense)
	if err != nil {
		return fmt.Sprintf("Error creating request for Splitwise API: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error POSTing expense to Splitwise API: %v", err)
	}
	defer resp.Body.Close()

	// read response
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Splitwise API call returned %d, but an error occurred reading the payload: %v", resp.StatusCode, err)
	}
	if !strings.Contains(string(b), `"expenses":[{`) {
		return fmt.Sprintf("Splitwise API call returned %d: %s", resp.StatusCode, string(b))
	}
	return "Expense successfully created on the Splitwise API."
}
