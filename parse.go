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
	spacesRegex = regexp.MustCompile(`\s+`)
	pricesRegex = regexp.MustCompile(`^(\d+)\.(\d{1,2})$`)
)

func parseReceiptTokens(s string) (tokens []string, priceIdxs []int) {
	priceIdxs = append(priceIdxs, -1)
	for _, tok := range spacesRegex.Split(s, -1) {
		if tok == "" {
			continue
		}
		if pricesRegex.MatchString(tok) {
			priceIdxs = append(priceIdxs, len(tokens))
		}
		tokens = append(tokens, tok)
	}
	priceIdxs = append(priceIdxs, len(tokens))
	return
}

func parsePriceToCents(tok string) priceInCents {
	m := pricesRegex.FindStringSubmatch(tok)
	euros, _ := strconv.ParseInt(m[1], 10, 64)
	if len(m[1]) == 0 {
		euros = 0
	}
	cents, _ := strconv.ParseInt(m[2], 10, 64)
	if len(m[2]) == 1 {
		cents *= 10
	}
	return priceInCents(cents + 100*euros)
}

func parseReceipt(s string) (receiptItems []*receiptItem) {
	tokens, priceIdxs := parseReceiptTokens(s)
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

func (p priceInCents) Format() string {
	mod := p % 100
	s := fmt.Sprintf("%d.%d", p/100, mod)
	if mod < 10 {
		s += "0"
	}
	return s
}
