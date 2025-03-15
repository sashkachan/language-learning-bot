package telegram

import (
	"context"
	"database/sql"
	"language-learning-bot/pkg/bot"
	"language-learning-bot/pkg/storage"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sashabaranov/go-openai"
)

func StartTelegramBot() {
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
		tgbotapi.BotCommand{Command: "inflection", Description: "Give inflection of a given word"},
		tgbotapi.BotCommand{Command: "translation", Description: "Provide translation of a phrase or a word"},
		tgbotapi.BotCommand{Command: "examples", Description: "Provide 3-4 examples of a word or a phrase"},
		tgbotapi.BotCommand{Command: "pronunciation", Description: "Pronounce a word or a phrase"},
		tgbotapi.BotCommand{Command: "speech_speed", Description: "Set speech speed"},
		tgbotapi.BotCommand{Command: "healthz", Description: "Check service health status"},
	)

	_, err = tgbot.Request(tgbotConfig)
	if err != nil {
		log.Fatal("Error setting commands:", err)
	}

	openaiClient := openai.NewClient(os.Getenv("OPENAI_API_TOKEN"))

	db, err := sql.Open("sqlite3", os.Getenv("SQLITE_PATH"))
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()

	initDBSQL, err := os.ReadFile("scripts/init_db.sql")
	if err != nil {
		log.Fatal("Error reading init_db.sql:", err)
	}

	_, err = db.Exec(string(initDBSQL))
	if err != nil {
		log.Fatal("Error executing init_db.sql:", err)
	}

	allowedUsers := []int64{}
	allowedUsersStr := os.Getenv("ALLOWED_TELEGRAM_USER_IDS")

	for _, id := range strings.Split(allowedUsersStr, ",") {

		allowedUser, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		allowedUsers = append(allowedUsers, allowedUser)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgbot.GetUpdatesChan(u)

	ScheduleQueriesRemoval(db)

	log.Println("Running...")

	for update := range updates {

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
				bot.HandleCallbackQuery(tgbot, openaiClient, update.CallbackQuery, db)
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
