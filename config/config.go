package config

import (
	"fmt"
	"os"

	"github.com/matheuscscp/splitwiser/models"

	"gopkg.in/yaml.v3"
)

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

// Load ...
func Load() (*Config, error) {
	confFile := os.Getenv("CONF_FILE")
	if confFile == "" {
		confFile = "config.yml"
	}
	b, err := os.ReadFile(confFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file '%s': %w", confFile, err)
	}
	var conf Config
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}
	return &conf, nil
}

// GetUserID ...
func (s *Splitwise) GetUserID(user models.ReceiptItemOwner) int64 {
	if user == models.Matheus {
		return s.MatheusID
	}
	return s.AnaID
}
