package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type (
	// Receipt ...
	Receipt []*ReceiptItem

	// ReceiptItem ...
	ReceiptItem struct {
		MainLine string
		NextLine string
		Price    PriceInCents
		Owner    ReceiptItemOwner
	}

	// PriceInCents ...
	PriceInCents int

	// ReceiptItemOwner ...
	ReceiptItemOwner string
)

const (
	Ana            ReceiptItemOwner = "a"
	Matheus        ReceiptItemOwner = "m"
	Shared         ReceiptItemOwner = "s"
	NotReceiptItem ReceiptItemOwner = "n"

	zeroCents PriceInCents = 0
)

var (
	regexSpaces                  = regexp.MustCompile(`\s+`)
	regexPriceToken              = regexp.MustCompile(`^(EUR|€){0,1}([0-9oO]+)\.([0-9oO]{1,2})$`)
	regexPriceOAsZero            = regexp.MustCompile(`[oO]`)
	regexTescoSingleAsteriskLine = regexp.MustCompile(`^\s*\*\s*$`)
)

// ParseReceipt ...
func ParseReceipt(receiptText string) (receipt Receipt) {
	lines := strings.Split(receiptText, "\n")
	if len(lines) == 1 {
		return parseLidlReceipt(receiptText)
	} else {
		return parseItemListFollowedByPriceList(lines)
	}
}

func parseLidlReceipt(receiptText string) (receipt Receipt) {
	tokens, priceIdxs := parseLidlReceiptTokens(receiptText)
	for i := 1; i < len(priceIdxs)-1; i++ {
		prevPriceIdx := priceIdxs[i-1]
		priceIdx := priceIdxs[i]
		nextPriceIdx := priceIdxs[i+1]
		mainLineToks := tokens[prevPriceIdx+1 : priceIdx]
		nextLineToks := tokens[priceIdx+1 : nextPriceIdx]
		if nextPriceIdx < len(tokens) {
			nextLineToks = append(nextLineToks, tokens[nextPriceIdx])
		}
		receipt = append(receipt, &ReceiptItem{
			MainLine: strings.Join(mainLineToks, " "),
			NextLine: strings.Join(nextLineToks, " "),
			Price:    parsePriceToCents(tokens[priceIdx]),
		})
	}
	return
}

func parseLidlReceiptTokens(receipt string) (tokens []string, priceIdxs []int) {
	priceIdxs = append(priceIdxs, -1)
	for _, tok := range regexSpaces.Split(receipt, -1) {
		if tok == "" {
			continue
		}
		if regexPriceToken.MatchString(tok) {
			priceIdxs = append(priceIdxs, len(tokens))
		}
		tokens = append(tokens, tok)
	}
	priceIdxs = append(priceIdxs, len(tokens))
	return
}

func parseItemListFollowedByPriceList(receiptLines []string) (receipt Receipt) {
	receiptLines = removeTescoSingleAsteriskLines(receiptLines)
	n := len(receiptLines) / 2
	receipt = make(Receipt, n)
	for i := 0; i < n; i++ {
		receipt[i] = &ReceiptItem{
			MainLine: receiptLines[i],
			Price:    parsePriceToCents(receiptLines[i+n]),
		}
	}
	return
}

func removeTescoSingleAsteriskLines(receiptLines []string) []string {
	var ret []string
	for _, s := range receiptLines {
		if !regexTescoSingleAsteriskLine.MatchString(s) {
			ret = append(ret, s)
		}
	}
	return ret
}

// ComputeTotals ...
func (r Receipt) ComputeTotals() (map[ReceiptItemOwner]PriceInCents, PriceInCents) {
	totalsInCents := make(map[ReceiptItemOwner]PriceInCents)
	var totalInCents PriceInCents
	for _, item := range r {
		if item.Owner == Ana || item.Owner == Matheus || item.Owner == Shared {
			totalsInCents[item.Owner] += item.Price
			totalInCents += item.Price
		}
	}
	return totalsInCents, totalInCents
}

// ComputeExpenses ...
func (r Receipt) ComputeExpenses(payer ReceiptItemOwner) (
	nonSharedExpense *Expense,
	sharedExpense *Expense,
) {
	totalsInCents, _ := r.ComputeTotals()

	cost, borrower, description := totalsInCents[Ana], Ana, "vegan"
	if payer == Ana {
		cost, borrower, description = totalsInCents[Matheus], Matheus, "non-vegan"
	}
	nonSharedExpense = &Expense{
		Cost: cost,
		UserShares: [2]*UserShare{
			{
				User: payer,
				Paid: cost,
				Owed: zeroCents,
			},
			{
				User: borrower,
				Paid: zeroCents,
				Owed: cost,
			},
		},
		Description: description,
	}

	costShared := totalsInCents[Shared]
	halfCostSharedRoundedDown := costShared / 2
	halfCostSharedRoundedUp := (costShared + 1) / 2
	sharedExpense = &Expense{
		Cost: costShared,
		UserShares: [2]*UserShare{
			{
				User: payer,
				Paid: costShared,
				Owed: halfCostSharedRoundedDown,
			},
			{
				User: borrower,
				Paid: zeroCents,
				Owed: halfCostSharedRoundedUp,
			},
		},
		Description: "shared",
	}

	return
}

// Len ...
func (r Receipt) Len() int {
	return len(r)
}

// NextItem ...
func (r Receipt) NextItem(curItem int) int {
	return (curItem + 1) % r.Len()
}

func parsePriceToCents(tok string) PriceInCents {
	m := regexPriceToken.FindStringSubmatch(tok)
	eurosStr, centsStr := m[2], m[3]
	eurosStr = regexPriceOAsZero.ReplaceAllString(eurosStr, "0")
	centsStr = regexPriceOAsZero.ReplaceAllString(centsStr, "0")
	euros, _ := strconv.ParseInt(eurosStr, 10, 64)
	if len(eurosStr) == 0 {
		euros = 0
	}
	cents, _ := strconv.ParseInt(centsStr, 10, 64)
	if len(centsStr) == 1 {
		cents *= 10
	}
	return PriceInCents(cents + 100*euros)
}

// String ...
func (p PriceInCents) String() string {
	s := fmt.Sprintf("%d.", p/100)
	mod := p % 100
	if mod < 10 {
		s += "0"
	}
	s += fmt.Sprintf("%d", mod)
	return s
}
