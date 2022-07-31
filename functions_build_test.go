package splitwiser

import (
	"testing"
)

func TestBuild(t *testing.T) {
	startBot := StartBot
	bot := Bot
	if startBot == nil || bot == nil {
		t.Fail()
	}
}
