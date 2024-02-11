package models_test

import (
	"testing"

	"github.com/matheuscscp/splitwiser/models"
	"github.com/stretchr/testify/assert"
)

func TestParsePriceInCents(t *testing.T) {
	for _, tt := range []struct {
		text     string
		expected models.PriceInCents
	}{
		{text: "-4", expected: -400},
		{text: "4", expected: 400},
		{text: ".4", expected: 40},
		{text: ".40", expected: 40},
		{text: "1.4", expected: 140},
		{text: "1.40", expected: 140},
		{text: "-.1", expected: -10},
		{text: "-0.1", expected: -10},
	} {
		t.Run(tt.text, func(t *testing.T) {
			tt := tt
			t.Parallel()

			actual := models.MustParsePriceInCents(tt.text)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
