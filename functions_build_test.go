package splitwiser

import (
	"testing"
)

func TestBuild(t *testing.T) {
	startBot := StartBot
	bot := Bot
	rotateJWTSecret := RotateJWTSecret
	if startBot == nil || bot == nil || rotateJWTSecret == nil {
		t.Fail()
	}
}
