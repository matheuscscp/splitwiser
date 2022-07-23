package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type (
	priceInCents int
	receiptItem  struct {
		mainLine     string
		nextLine     string
		priceInCents priceInCents
	}
)

var (
	spacesRegex                  = regexp.MustCompile(`\s+`)
	priceTokenRegex              = regexp.MustCompile(`^(EUR){0,1}(\d+)\.(\d{1,2})$`)
	tescoSingleAsteriskLineRegex = regexp.MustCompile(`^\s*\*\s*$`)
)

func parsePriceToCents(tok string) priceInCents {
	m := priceTokenRegex.FindStringSubmatch(tok)
	euros, _ := strconv.ParseInt(m[2], 10, 64)
	if len(m[2]) == 0 {
		euros = 0
	}
	cents, _ := strconv.ParseInt(m[3], 10, 64)
	if len(m[3]) == 1 {
		cents *= 10
	}
	return priceInCents(cents + 100*euros)
}

func (p priceInCents) Format() string {
	s := fmt.Sprintf("%d.", p/100)
	mod := p % 100
	if mod < 10 {
		s += "0"
	}
	s += fmt.Sprintf("%d", mod)
	return s
}

func parseReceipt(receipt string) (receiptItems []*receiptItem) {
	lines := strings.Split(receipt, "\n")
	if len(lines) == 1 {
		return parseLidlReceipt(receipt)
	} else {
		return parseItemListFollowedByPriceList(lines)
	}
}

func parseLidlReceipt(receipt string) (receiptItems []*receiptItem) {
	tokens, priceIdxs := parseLidlReceiptTokens(receipt)
	for i := 1; i < len(priceIdxs)-1; i++ {
		prevPriceIdx := priceIdxs[i-1]
		priceIdx := priceIdxs[i]
		nextPriceIdx := priceIdxs[i+1]
		mainLineToks := tokens[prevPriceIdx+1 : priceIdx]
		nextLineToks := tokens[priceIdx+1 : nextPriceIdx]
		if nextPriceIdx < len(tokens) {
			nextLineToks = append(nextLineToks, tokens[nextPriceIdx])
		}
		receiptItems = append(receiptItems, &receiptItem{
			mainLine:     strings.Join(mainLineToks, " "),
			nextLine:     strings.Join(nextLineToks, " "),
			priceInCents: parsePriceToCents(tokens[priceIdx]),
		})
	}
	return
}

func parseLidlReceiptTokens(receipt string) (tokens []string, priceIdxs []int) {
	priceIdxs = append(priceIdxs, -1)
	for _, tok := range spacesRegex.Split(receipt, -1) {
		if tok == "" {
			continue
		}
		if priceTokenRegex.MatchString(tok) {
			priceIdxs = append(priceIdxs, len(tokens))
		}
		tokens = append(tokens, tok)
	}
	priceIdxs = append(priceIdxs, len(tokens))
	return
}

func parseItemListFollowedByPriceList(receiptLines []string) (receiptItems []*receiptItem) {
	receiptLines = removeTescoSingleAsteriskLines(receiptLines)
	n := len(receiptLines) / 2
	receiptItems = make([]*receiptItem, n)
	for i := 0; i < n; i++ {
		receiptItems[i] = &receiptItem{
			mainLine:     receiptLines[i],
			priceInCents: parsePriceToCents(receiptLines[i+n]),
		}
	}
	return
}

func removeTescoSingleAsteriskLines(receiptLines []string) []string {
	var ret []string
	for _, s := range receiptLines {
		if !tescoSingleAsteriskLineRegex.MatchString(s) {
			ret = append(ret, s)
		}
	}
	return ret
}
