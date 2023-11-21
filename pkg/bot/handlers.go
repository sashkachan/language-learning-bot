package bot

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"os"
	"strings"

	openai_api "language-learning-bot/pkg/openai"
	storage "language-learning-bot/pkg/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

// ... [imports] ...

func HandleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		// ... [handle start command] ...
		sendLanguageSelection(bot, message.Chat.ID)

	case "settings":
		// ... [handle settings command] ...
	}
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *sql.DB) {
	// ... [callback query handling logic] ...
	data := callbackQuery.Data
	if strings.HasPrefix(data, "language:") {
		language := strings.Split(data, ":")[1]
		updateLanguagePreference(bot, callbackQuery, db, language)
		// ...
	}
}

func sendLanguageSelection(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Please choose a language:")
	msg.ReplyMarkup = languageInlineKeyboard()
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending language selection: %v\n", err)
	}
}

func languageInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Dutch", "language:Dutch"),
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

	// Define the template for the GPT prompt
	gptTemplateWordExamples := os.Getenv("GPT_TEMPLATE_WORD_EXAMPLES")

	// Create a data structure to hold the template variables
	data := struct {
		Language    string
		MessageText string
	}{
		Language:    language,
		MessageText: message.Text,
	}

	// Create a new template and parse the template string
	tmpl := template.New("gptTemplate")
	tmpl, err = tmpl.Parse(gptTemplateWordExamples)
	if err != nil {
		log.Printf("Error parsing GPT template: %v\n", err)
		return
	}

	// Execute the template with the data
	var gptPrompt strings.Builder
	err = tmpl.Execute(&gptPrompt, data)
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

	// ... [handle message] ...
	msg := tgbotapi.NewMessage(message.Chat.ID, gptresponse)
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("Error sending GPT response: %v\n", err)
	}
}

// IsAllowedUser checks if user is allowed to use bot
func IsAllowedUser(update tgbotapi.Update, allowedUsers []int64) bool {
	// check if user is allowed to use bot
	userID := update.Message.From.ID
	for _, allowedUser := range allowedUsers {
		if userID == allowedUser {
			return true
		}
	}
	return false
}
