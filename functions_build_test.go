package splitwiser_test

import (
	"testing"

	"github.com/matheuscscp/splitwiser"
)

func TestBuild(t *testing.T) {
	startBot := splitwiser.StartBot
	bot := splitwiser.Bot
	if startBot == nil || bot == nil {
		t.Fail()
	}
}
