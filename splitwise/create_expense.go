package splitwise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/matheuscscp/splitwiser/config"
	"github.com/matheuscscp/splitwiser/models"
)

type (
	Client interface {
		// CreateExpense creates an expense on the Splitwise API and
		// returns a message for the bot to send back as result.
		CreateExpense(expense *models.Expense, storeName string) (msg string)
	}

	client struct {
		conf *config.Splitwise
	}
)

// NewClient ...
func NewClient(conf *config.Splitwise) Client {
	return &client{conf: conf}
}

func (c *client) CreateExpense(expense *models.Expense, storeName string) string {
	if expense.Cost < 0 {
		return "Skipping expense with negative cost."
	}

	if expense.Cost == 0 {
		return "Skipping expense with cost zero."
	}

	// create payload
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(map[string]interface{}{
		"currency_code":        "EUR",
		"category_id":          12, // Groceries
		"description":          fmt.Sprintf("%s %s", storeName, expense.Description),
		"cost":                 expense.Cost.String(),
		"group_id":             c.conf.GroupID,
		"users__0__user_id":    c.conf.GetUserID(expense.UserShares[0].User),
		"users__0__paid_share": expense.UserShares[0].Paid.String(),
		"users__0__owed_share": expense.UserShares[0].Owed.String(),
		"users__1__user_id":    c.conf.GetUserID(expense.UserShares[1].User),
		"users__1__paid_share": expense.UserShares[1].Paid.String(),
		"users__1__owed_share": expense.UserShares[1].Owed.String(),
	})
	if err != nil {
		return fmt.Sprintf("Error encoding Splitwise JSON body: %v", err)
	}

	// make request
	req, err := http.NewRequest(http.MethodPost, "https://secure.splitwise.com/api/v3.0/create_expense", &buf)
	if err != nil {
		return fmt.Sprintf("Error creating request for Splitwise API: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.conf.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error POSTing expense to Splitwise API: %v", err)
	}
	defer resp.Body.Close()

	// check response
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Splitwise API call returned %d, but an error occurred reading the payload: %v", resp.StatusCode, err)
	}
	if !strings.Contains(string(b), `"expenses":[{`) {
		return fmt.Sprintf("Splitwise API call returned %d: %s", resp.StatusCode, string(b))
	}

	return "Expense successfully created on the Splitwise API."
}
