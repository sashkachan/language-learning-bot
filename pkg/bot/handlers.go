package bot

import (
	"context"
	"database/sql"
	"text/template"

	"log"
	"strings"

	"language-learning-bot/pkg/config"
	openai_api "language-learning-bot/pkg/openai"
	storage "language-learning-bot/pkg/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

func HandleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) error {
	// log the command to the console
	log.Printf("%d [%s] %s", message.From.ID, message.From.UserName, message.Text)
	response := ""
	switch message.Command() {
	case "start":
		if err := sendLanguageSelection(bot, message.Chat.ID); err != nil {
			log.Printf("Error sending language selection: %v\n", err)
			return err
		}
		response = ""

	case "examples":
		if err := handleExamplesCommand(bot, message, db); err != nil {
			log.Printf("Error handling examples command: %v\n", err)
			return err
		}
		response = "I will respond with examples of the word or phrase usage."

	case "translation":
		if err := handleTranslationCommand(bot, message, db); err != nil {
			log.Printf("Error handling translation command: %v\n", err)
			return err
		}
		response = "I will respond with translations."
	}
	// send the response to the user
	if response != "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending response: %v\n", err)
			return err
		}
	}
	return nil
}

func handleExamplesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) error {
	return storage.UpdateUserHelpType(db, int(message.From.ID), "examples")
}

func handleTranslationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) error {
	return storage.UpdateUserHelpType(db, int(message.From.ID), "translation")
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *sql.DB) {
	data := callbackQuery.Data
	if strings.HasPrefix(data, "language:") {
		language := strings.Split(data, ":")[1]
		updateLanguagePreference(bot, callbackQuery, db, language)
	}
}

func sendLanguageSelection(bot *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Please choose a language:")
	msg.ReplyMarkup = languageInlineKeyboard()
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending language selection: %v\n", err)
		return err
	}
	return nil
}

func languageInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Dutch", "language:Dutch"),
			tgbotapi.NewInlineKeyboardButtonData("French", "language:French"),
			tgbotapi.NewInlineKeyboardButtonData("German", "language:German"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Estonian", "language:Estonian"),
			tgbotapi.NewInlineKeyboardButtonData("Spanish", "language:Spanish"),
			tgbotapi.NewInlineKeyboardButtonData("Russian", "language:Russian"),
		),
	)
	return keyboard
}

func updateLanguagePreference(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *sql.DB, language string) {
	userID := int(callbackQuery.From.ID)
	err := storage.UpdateUserLanguage(db, userID, language)
	if err != nil {
		// Handle error
		log.Printf("Error updating language preference: %v\n", err)
		return
	}

	// Send a confirmation message and remove the inline keyboard
	msg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, "Language set to "+language)
	// msg.ReplyMarkup = &emptyKeyboard
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("Error sending confirmation message: %v\n", err)
	}
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, openaiClient *openai.Client, db *sql.DB) {
	userID := int(message.From.ID)
	language, err := storage.GetUserLanguage(db, userID)
	if err != nil {
		// Handle error
		log.Printf("Error getting user language: %v\n", err)
		return
	}

	helpType, shouldReturn := GetUserHelpType(db, userID)
	if shouldReturn {
		return
	}

	// Fetch the examples and translation templates using NewConfig
	config := config.NewConfig()

	// Determine the template to use based on the user setting
	var gptTemplate *template.Template
	switch helpType {
	case "examples":
		gptTemplate = config.GptTemplateWordUsageExamples
	case "translation":
		gptTemplate = config.GptTemplateWordTranslation
	default:
		log.Printf("Invalid help type: %s\n", helpType)
		return
	}

	// Create a data structure to hold the template variables
	data := struct {
		Language    string
		MessageText string
	}{
		Language:    language,
		MessageText: message.Text,
	}

	// Execute the template with the data
	var gptPrompt strings.Builder
	err = gptTemplate.Execute(&gptPrompt, data)
	if err != nil {
		log.Printf("Error executing GPT template: %v\n", err)
		return
	}

	// create new context
	ctx := context.Background()

	gptresponse, err := openai_api.GetGPTResponse(ctx, openaiClient, gptPrompt.String())
	if err != nil {
		log.Printf("Error getting GPT response: %v\n", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, gptresponse)
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("Error sending GPT response: %v\n", err)
	}
}

func GetUserHelpType(db *sql.DB, userID int) (string, bool) {
	helpType, err := storage.GetUserHelpType(db, userID)
	if err != nil {

		log.Printf("Error getting user help_type: %v\n", err)
		return "", true
	}
	if helpType == "" {
		helpType = "examples"
		err = storage.UpdateUserHelpType(db, userID, helpType)
		if err != nil {

			log.Printf("Error updating user help_type: %v\n", err)
			return "", true
		}
	}
	return helpType, false
}

// IsAllowedUser checks if user is allowed to use bot
func IsAllowedUser(update tgbotapi.Update, allowedUsers []int64) bool {
	// check if user is allowed to use bot. Try update.Message.From.ID first, then update.CallbackQuery.From.ID
	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	} else {
		return false
	}
	for _, allowedUser := range allowedUsers {
		if userID == allowedUser {
			return true
		}
	}
	return false
}
