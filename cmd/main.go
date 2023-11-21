package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	bot "language-learning-bot/pkg/bot"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	openai "github.com/sashabaranov/go-openai"
)

func pretty_print(update interface{}) {
	b, err := json.MarshalIndent(update, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(b))
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	tgbot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	// Initialize OpenAI Client
	openaiClient := openai.NewClient(os.Getenv("OPENAI_API_TOKEN"))

	// Initialize SQLite Database
	db, err := sql.Open("sqlite3", os.Getenv("SQLITE_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Start listening for updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgbot.GetUpdatesChan(u)

	log.Println("Running...")
	// Handle updates (commands, messages)
	for update := range updates {
		pretty_print(update)
		if update.Message != nil {
			if update.Message.IsCommand() {
				bot.HandleCommand(tgbot, update.Message)
			} else {
				bot.HandleMessage(tgbot, update.Message, openaiClient, db)
			}
		} else if update.CallbackQuery != nil {
			bot.HandleCallbackQuery(tgbot, update.CallbackQuery, db)
		}
	}
}
