package main

import (
	"os"
	"testing"
)

func TestMainSetup(t *testing.T) {
	// Check if environment variables are set
	if os.Getenv("TELEGRAM_TOKEN") == "" || os.Getenv("OPENAI_TOKEN") == "" {
		t.Error("Environment variables not set")
	}

	// You can add more tests here to check the setup
}
