package splitwiser

import (
	"testing"
)

func TestBuild(t *testing.T) {
	startBot := StartBot
	bot := Bot
	rotateSecret := RotateSecret
	if startBot == nil || bot == nil || rotateSecret == nil {
		t.Fail()
	}
}
