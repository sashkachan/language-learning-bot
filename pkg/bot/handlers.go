package bot

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"language-learning-bot/pkg/config"
	openai_api "language-learning-bot/pkg/openai"
	storage "language-learning-bot/pkg/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

// HandleCommand handles the incoming command from the user and performs the corresponding action.
// It logs the command to the console, switches based on the command type, and sends a response back to the user.
// Parameters:
// - bot: A pointer to the tgbotapi.BotAPI instance.
// - message: A pointer to the tgbotapi.Message instance representing the incoming message.
// - db: A pointer to the sql.DB instance for database operations.
// - openaiClient: A pointer to the openai.Client instance for OpenAI API operations.
// Returns:
// - An error if any error occurs during the handling of the command, otherwise nil.
func HandleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
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
		if err := handleExamplesCommand(bot, message, db, openaiClient); err != nil {
			log.Printf("Error handling examples command: %v\n", err)
			return err
		}
		response = "I will respond with examples of the word or phrase usage."

	case "translation":
		if err := handleTranslationCommand(bot, message, db, openaiClient); err != nil {
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

func handleExamplesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	// get the last interaction so we can send the user the last word they queried
	lastQuery, err := storage.GetLastUserQuery(db, int(message.Chat.ID))
	if err != nil {
		log.Printf("Error getting last query: %v\n", err)
		return err
	}

	// get user language
	language, err := storage.GetUserLanguage(db, int(message.From.ID))
	if err != nil {
		log.Printf("Error getting user language: %v\n", err)
		return err
	}

	// log last query
	log.Printf("Last query: %v\n", lastQuery)
	// check if the last query type is not already examples. If it is, do nothing, otherwise
	// query gpt api for examples
	if lastQuery.Type != "examples" {
		response, err := ProcessQuery("examples", language, lastQuery.Word, db, int(message.Chat.ID), openaiClient)
		if err != nil {
			log.Printf("Error processing query: %v\n", err)
			return err
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		_, err = bot.Send(msg)
		if err != nil {
			log.Printf("Error sending GPT response: %v\n", err)
			return err
		}
	}
	return storage.UpdateUserHelpType(db, int(message.From.ID), "examples")
}

func handleTranslationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	// get the last interaction so we can send the user the last word they queried
	lastQuery, err := storage.GetLastUserQuery(db, int(message.Chat.ID))
	if err != nil {
		log.Printf("Error getting last query: %v\n", err)
		return err
	}

	// get user language
	language, err := storage.GetUserLanguage(db, int(message.From.ID))
	if err != nil {
		log.Printf("Error getting user language: %v\n", err)
		return err
	}

	// log last query
	log.Printf("Last query: %v\n", lastQuery)
	// check if the last query type is not already translation. If it is, do nothing, otherwise
	// query gpt api for translation
	if lastQuery.Type != "translation" {
		response, err := ProcessQuery("translation", language, lastQuery.Word, db, int(message.Chat.ID), openaiClient)
		if err != nil {
			log.Printf("Error processing query: %v\n", err)
			return err
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		_, err = bot.Send(msg)
		if err != nil {
			log.Printf("Error sending GPT response: %v\n", err)
			return err
		}
	}
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

type GptTemplateData struct {
	Language    string
	MessageText string
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, openaiClient *openai.Client, db *sql.DB) {
	userID := int(message.From.ID)
	language, err := storage.GetUserLanguage(db, userID)
	if err != nil {
		// Handle error
		log.Printf("Error getting user language: %v\n", err)
		return
	}

	helpType, err := GetUserHelpType(db, userID)
	if err != nil {
		return
	}

	gptresponse, err := ProcessQuery(helpType, language, message.Text, db, userID, openaiClient)
	if err != nil {
		log.Printf("Error processing query: %v\n", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, gptresponse)
	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("Error sending GPT response: %v\n", err)
	}
}

// ProcessQuery processes a query based on the given parameters.
// It checks if a cached response exists for the query and returns it if found.
// If no cached response is found, it generates a response using the GPT model.
// The generated response is then cached for future use.
//
// Parameters:
// - helpType: The type of help requested (e.g., "examples", "translation").
// - language: The language of the query.
// - message: The query message.
// - db: The database connection.
// - userID: The ID of the user making the query.
// - openaiClient: The OpenAI client for generating GPT responses.
//
// Returns:
// - string: The generated response or the cached response.
// - error: An error if any occurred during the process.
func ProcessQuery(helpType string, language string, message string, db *sql.DB, userID int, openaiClient *openai.Client) (string, error) {
	gptConfig := config.NewConfig()

	// check if we can find cached response
	log.Printf("Checking cache for response: language=%s, type=%s, word=%s\n", language, helpType, message)

	cachedResponse, err := storage.GetCachedResponseByWordAndType(db, language, helpType, message)
	if err != nil {
		log.Printf("Error getting cached response: %v\n", err)
		return "", err
	}

	if cachedResponse != "" {
		log.Printf("Found cached response")
		// store query
		log.Printf("Storing query: userID=%d, message=%s\n", userID, message)
		_, err := storage.StoreQuery(db, userID, helpType, language, message)
		if err != nil {
			log.Printf("Error storing query: %v\n", err)
		}
		return cachedResponse, nil
	}

	var gpt *config.GptRequestType
	switch helpType {
	case "examples":
		gpt = gptConfig.GptTemplateWordUsageExamples
	case "translation":
		gpt = gptConfig.GptTemplateWordTranslation
	default:
		log.Printf("invalid help type: %s\n", helpType)
		return "", errors.New("invalid help type")
	}

	data := GptTemplateData{
		Language:    language,
		MessageText: message,
	}

	var gptPrompt strings.Builder
	err = gpt.PromptTemplate.Execute(&gptPrompt, data)
	if err != nil {
		log.Printf("Error executing GPT template: %v\n", err)
		return "", err
	}

	// log storing query: userID, message
	log.Printf("Storing query: %d, %s\n", userID, message)
	query_id, err := storage.StoreQuery(db, userID, helpType, language, message)
	if err != nil {
		log.Printf("Error storing query: %v\n", err)
	}

	gptRequest := openai_api.GPTRequest{
		Prompt:                 gptPrompt.String(),
		WordOrPhrase:           message,
		ChatCompletionMessages: gptConfig.GptPromptTunings[language][helpType].Messages,
	}

	ctx := context.Background()

	gptresponse, err := openai_api.GetGPTResponse(ctx, openaiClient, gptRequest)
	if err != nil {
		log.Printf("Error getting GPT response: %v\n", err)
		return "", err
	}

	// cache response
	log.Printf("Caching response: language=%s, type=%s, word=%s\n", language, helpType, message)
	err = storage.CacheResponse(db, query_id, gptresponse)
	if err != nil {
		log.Printf("Error caching response: %v\n", err)
		return "", err
	}
	return gptresponse, nil
}

func GetUserHelpType(db *sql.DB, userID int) (string, error) {
	helpType, err := storage.GetUserHelpType(db, userID)
	if err != nil {

		log.Printf("Error getting user help_type: %v\n", err)
		return "", err
	}
	if helpType == "" {
		helpType = "examples"
		err = storage.UpdateUserHelpType(db, userID, helpType)
		if err != nil {

			log.Printf("Error updating user help_type: %v\n", err)
			return "", err
		}
	}
	return helpType, nil
}

// IsAllowedUser checks if user is allowed to use bot
func IsAllowedUser(update tgbotapi.Update, allowedUsers []int64) bool {
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
