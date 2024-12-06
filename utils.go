package main

import (
	"encoding/json"
	"os"
	"time"
)

// Configuration filled from configuration file
type Configuration struct {
	DiscordBotKey string
	ApiID         string
	TimeOut       time.Duration
	MaxChars      int
	MaxSentences  int
	WarningChars  int
}

// loadConfiguration loads configuration from json file
func loadConfiguration(configurationFile string) (Configuration, error) {
	configuration := Configuration{}

	file, err := os.Open(configurationFile)
	if err != nil {
		return configuration, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	return configuration, err
}
