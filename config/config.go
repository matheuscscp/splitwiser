package config

import "github.com/matheuscscp/splitwiser/models"

type (
	// Config ...
	Config struct {
		Telegram struct {
			Token  string `yaml:"token"`
			ChatID int64  `yaml:"chatID"`
		} `yaml:"telegram"`
		Splitwise        Splitwise `yaml:"splitwise"`
		CheckpointBucket string    `yaml:"checkpointBucket"`
	}

	// Splitwise ...
	Splitwise struct {
		Token     string `yaml:"token"`
		GroupID   int64  `yaml:"groupID"`
		AnaID     int64  `yaml:"anaID"`
		MatheusID int64  `yaml:"matheusID"`
	}
)

func (s *Splitwise) GetUserID(user models.ReceiptItemOwner) int64 {
	if user == models.Matheus {
		return s.MatheusID
	}
	return s.AnaID
}
