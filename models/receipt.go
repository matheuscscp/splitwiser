package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type (
	Receipt []*ReceiptItem

	ReceiptItem struct {
		Name  string           `json:"name"`
		Price PriceInCents     `json:"euro_cents"`
		Owner ReceiptItemOwner `json:"owner"`
	}

	PriceInCents int

	ReceiptItemOwner string
)

const (
	Ana     ReceiptItemOwner = "a"
	Matheus ReceiptItemOwner = "m"
	Shared  ReceiptItemOwner = "s"

	zeroCents PriceInCents = 0
)

var (
	regexSpaces     = regexp.MustCompile(`\s+`)
	regexPriceToken = regexp.MustCompile(`^\s*(-{0,1})([0-9]*)((\.([0-9]{1,2})){0,1})\s*$`)
)

func ParseReceipt(receiptText string) (receipt Receipt) {
	var tokens []string
	priceIdxs := []int{-1}
	for _, tok := range regexSpaces.Split(receiptText, -1) {
		if tok == "" {
			continue
		}
		if regexPriceToken.MatchString(tok) {
			priceIdxs = append(priceIdxs, len(tokens))
		}
		tokens = append(tokens, tok)
	}
	for i := 1; i < len(priceIdxs); i++ {
		priceIdx := priceIdxs[i]
		nameTokens := tokens[priceIdxs[i-1]+1 : priceIdx]
		receipt = append(receipt, &ReceiptItem{
			Name:  strings.Join(nameTokens, " "),
			Price: MustParsePriceInCents(tokens[priceIdx]),
		})
	}
	return
}

func (r Receipt) AbsoluteTotal() (total PriceInCents) {
	for _, item := range r {
		total += item.Price
	}
	return
}

func (r Receipt) ComputeTotals() (ownerTotals map[ReceiptItemOwner]PriceInCents,
	total PriceInCents, totalWithDiscounts PriceInCents) {
	ownerTotals = make(map[ReceiptItemOwner]PriceInCents)
	for _, item := range r {
		if item.Owner == Ana || item.Owner == Matheus || item.Owner == Shared {
			ownerTotals[item.Owner] += item.Price
			totalWithDiscounts += item.Price
			if item.Price > 0 {
				total += item.Price
			}
		}
	}
	return
}

func (r Receipt) ComputeExpenses(payer ReceiptItemOwner) (
	nonSharedExpense *Expense,
	sharedExpense *Expense,
) {
	ownerTotals, _, _ := r.ComputeTotals()

	cost, borrower, description := ownerTotals[Ana], Ana, "vegan"
	if payer == Ana {
		cost, borrower, description = ownerTotals[Matheus], Matheus, "non-vegan"
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

	costShared := ownerTotals[Shared]
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

func (r Receipt) Len() int {
	return len(r)
}

func (r Receipt) NextItem(curItem int) int {
	return (curItem + 1) % r.Len()
}

func (r Receipt) String() string {
	items := make([]string, r.Len())
	for i := range r {
		items[i] = r[i].String()
	}
	items = append(items, "")
	items = append(items, fmt.Sprintf("Total: %s", r.AbsoluteTotal()))
	return strings.Join(items, "\n")
}

func (r *ReceiptItem) String() string {
	return fmt.Sprintf("%s (%s)", r.Name, r.Price)
}

func ParsePriceInCents(tok string) (PriceInCents, bool) {
	m := regexPriceToken.FindStringSubmatch(tok)
	if m == nil {
		return 0, false
	}
	sign, eurosStr, centsStr := m[1], m[2], m[5]
	euros, _ := strconv.ParseInt(eurosStr, 10, 64)
	if len(eurosStr) == 0 {
		euros = 0
	}
	cents, _ := strconv.ParseInt(centsStr, 10, 64)
	if len(centsStr) == 1 {
		cents *= 10
	}
	price := PriceInCents(cents + 100*euros)
	if sign == "-" {
		price = -price
	}
	return price, true
}

func MustParsePriceInCents(tok string) PriceInCents {
	price, ok := ParsePriceInCents(tok)
	if !ok {
		panic(fmt.Errorf("%q does not match pattern %q", tok, regexPriceToken.String()))
	}
	return price
}

func (p PriceInCents) String() string {
	i := int(p)
	var s string
	if i < 0 {
		i = -i
		s += "-"
	}
	s += fmt.Sprintf("%d.", i/100)
	mod := i % 100
	if mod < 10 {
		s += "0"
	}
	s += fmt.Sprintf("%d", mod)
	return s
}

func (o ReceiptItemOwner) Pretty() string {
	switch o {
	case Ana:
		return "Ana"
	case Matheus:
		return "Matheus"
	case Shared:
		return "Shared"
	default:
		return string(o)
	}
}
