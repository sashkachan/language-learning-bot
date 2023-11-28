package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	bot "language-learning-bot/pkg/bot"
	"language-learning-bot/pkg/storage"

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
		log.Printf("Error loading .env file: %v\n", err)
	}

	tgbot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	tgbotConfig := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Configure the preferred language"},
		tgbotapi.BotCommand{Command: "translation", Description: "Provide translation of a phrase or a word"},
		tgbotapi.BotCommand{Command: "examples", Description: "Provide 3-4 examples of a word or a phrase"},
	)
	_, err = tgbot.Request(tgbotConfig)
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

	// load the contents of scripts/init_db.sql into a string
	initDBSQL, err := os.ReadFile("scripts/init_db.sql")
	if err != nil {
		log.Fatal(err)
	}
	// execute the SQL query
	_, err = db.Exec(string(initDBSQL))
	if err != nil {
		log.Fatal(err)
	}

	// allowed telegram users ids
	allowedUsers := []int64{}
	allowedUsersStr := os.Getenv("ALLOWED_TELEGRAM_USER_IDS")
	// split string by comma
	for _, id := range strings.Split(allowedUsersStr, ",") {
		// convert string to int64
		allowedUser, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		allowedUsers = append(allowedUsers, allowedUser)
	}

	// Start listening for updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgbot.GetUpdatesChan(u)

	ScheduleQueriesRemoval(db)

	log.Println("Running...")
	// Handle updates (commands, messages)
	for update := range updates {
		// pretty_print(update)
		// check if user is allowed to use bot
		go func(update tgbotapi.Update) {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered in f", r)
				}
			}()

			ctx := context.Background()

			if !bot.IsAllowedUser(update, allowedUsers) {
				log.Printf("User %d is not allowed to use bot", update.Message.From.ID)
				return
			}
			if update.Message != nil {
				if update.Message.IsCommand() {
					err := bot.HandleCommand(ctx, tgbot, update.Message, db, openaiClient)
					if err != nil {
						log.Printf("Error handling command: %v\n", err)
					}

				} else {
					bot.HandleMessage(ctx, tgbot, update.Message, openaiClient, db)
				}
			} else if update.CallbackQuery != nil {
				bot.HandleCallbackQuery(tgbot, update.CallbackQuery, db)
			}
		}(update)
	}
}

func ScheduleQueriesRemoval(db *sql.DB) {
	// check if CACHE_CLEAN_INTERVAL_HOURS is set, otherwise set default value to 24
	cacheCleanIntervalHoursStr := os.Getenv("CACHE_CLEAN_INTERVAL_HOURS")
	if cacheCleanIntervalHoursStr == "" {
		cacheCleanIntervalHoursStr = "24"
	}
	cacheCleanIntervalHours, err := strconv.Atoi(cacheCleanIntervalHoursStr)
	if err != nil {
		log.Fatal(err)
	}
	// schedule queries removal
	ticker := time.NewTicker(time.Duration(cacheCleanIntervalHours) * time.Hour)

	defer ticker.Stop()
	go func() {
		for range ticker.C {
			err := storage.CleanOldCachedResponses(db)
			if err != nil {
				log.Println("Error cleaning old cached responses:", err)
			}
		}
	}()
}
