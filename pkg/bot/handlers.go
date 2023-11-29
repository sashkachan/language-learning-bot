package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
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
func HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
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

	case "pronunciation":
		if err := handlePronounciationCommand(bot, message, db, openaiClient); err != nil {
			log.Printf("Error handling pronounciation command: %v\n", err)
			return err
		}
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

func parseExamplesByNumber(message string) []string {
	// parse the last response for the strings starting with a number
	lines := strings.Split(message, "\n")
	var examples []string
	numberPrefixRegex := regexp.MustCompile(`[0-9]+\. `)
	for _, line := range lines {
		// check if the line matches the pattern "[0-9]+\. "
		if numberPrefixRegex.MatchString(line) {
			// strip the number from the line
			line = strings.Split(line, ". ")[1]
			// add the line to the examples
			examples = append(examples, line)
		}
	}
	return examples
}

func handlePronounciationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	userId := int(message.From.ID)
	sendLastRequestAudio(db, userId, 0, message.Text, openaiClient, bot)

	return nil
}

func sendAudioMessage(openaiClient *openai.Client, firstLine string, userid int, bot *tgbotapi.BotAPI) error {
	openaiResponse, err := openai_api.GetTTSResponse(context.Background(), openaiClient, firstLine)
	if err != nil {
		log.Printf("Error getting TTS response: %v\n", err)
		return err
	}

	audio := tgbotapi.FileBytes{Name: fmt.Sprintf("%s.mp3", firstLine), Bytes: openaiResponse}
	audioMsg := tgbotapi.NewAudio(int64(userid), audio)
	_, err = bot.Send(audioMsg)
	if err != nil {
		log.Printf("Error sending audio message: %v\n", err)
		return err
	}
	return nil
}

func handleExamplesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	err := storage.UpdateUserHelpType(db, int(message.From.ID), "examples")
	if err != nil {
		log.Printf("Error updating user help_type: %v\n", err)
		return err
	}
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
	return err
}

func handleTranslationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	err := storage.UpdateUserHelpType(db, int(message.From.ID), "translation")
	if err != nil {
		log.Printf("Error updating user help_type: %v\n", err)
		return err
	}
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
	return err
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, openaiClient *openai.Client, callbackQuery *tgbotapi.CallbackQuery, db *sql.DB) {
	data := callbackQuery.Data
	if strings.HasPrefix(data, "language:") {
		language := strings.Split(data, ":")[1]
		updateLanguagePreference(bot, callbackQuery, db, language)
	}

	if strings.HasPrefix(data, "pronunciation:") {
		// parse the number from the callback data into an int
		exampleNumber, err := strconv.Atoi(strings.Split(data, ":")[1])
		if err != nil {
			log.Printf("Error parsing example number: %v\n", err)
			return
		}
		log.Println("Pronounciation example: ", exampleNumber)
		userId := int(callbackQuery.From.ID)

		// send the Nth example
		shouldReturn := sendLastRequestAudio(db, userId, exampleNumber, callbackQuery.Message.Text, openaiClient, bot)
		if shouldReturn {
			log.Printf("Error sending last request audio")
			return
		}
		// Send a confirmation message and remove the inline keyboard
		msg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, fmt.Sprintf("Example %d sent", exampleNumber))
		// msg.ReplyMarkup = &emptyKeyboard
		_, err = bot.Send(msg)
		if err != nil {
			log.Printf("Error sending confirmation message: %v\n", err)
		}
	}
}

func sendLastRequestAudio(db *sql.DB, userId int, exampleNumber int, message string, openaiClient *openai.Client, bot *tgbotapi.BotAPI) bool {
	lastQuery, err := storage.GetLastUserQuery(db, userId)
	if err != nil {
		log.Printf("Error getting last query: %v\n", err)
		return true
	}
	log.Println(lastQuery)
	lastResponse, err := storage.GetCachedResponseByWordLangAndType(db, lastQuery.Language, lastQuery.Type, lastQuery.Word)

	if err != nil {
		log.Printf("Error getting cached response: %v\n", err)
		return true
	}

	if lastQuery.Type == "examples" {
		examples := parseExamplesByNumber(lastResponse)
		log.Printf("Examples: %v\n", examples)

		if exampleNumber == 0 {
			// draw the inline keyboard with the examples
			err := sendExamplesSelection(bot, int64(userId), len(examples))
			if err != nil {
				log.Printf("Error sending examples selection: %v\n", err)
				return true
			}
		} else if len(examples) > 0 && len(examples) >= exampleNumber {
			err := sendAudioMessage(openaiClient, examples[exampleNumber-1], userId, bot)
			if err != nil {
				log.Printf("Error sending audio message: %v\n", err)
				return true
			}
		}
	} else if lastQuery.Type == "translation" {
		lastResponseLines := strings.Split(lastResponse, "\n")
		if len(lastResponseLines) > 0 {
			firstLine := lastResponseLines[0]
			log.Printf("First line: %s\n", firstLine)

			err := sendAudioMessage(openaiClient, firstLine, userId, bot)
			if err != nil {
				log.Printf("Error sending audio message: %v\n", err)
				return true
			}
		}
	}

	return false
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

func sendExamplesSelection(bot *tgbotapi.BotAPI, chatID int64, total int) error {
	msg := tgbotapi.NewMessage(chatID, "Please choose an example:")
	msg.ReplyMarkup = examplesInlineKeyboard(total)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending examples selection: %v\n", err)
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

func examplesInlineKeyboard(total int) tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	currentInlineRow := tgbotapi.NewInlineKeyboardRow()

	for i := 1; i <= total; i++ {
		if i%3 == 0 {
			// add the current row to the keyboard
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, currentInlineRow)
			currentInlineRow = tgbotapi.NewInlineKeyboardRow()
		}
		currentInlineRow = append(currentInlineRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", i), fmt.Sprintf("pronunciation:%d", i)))
		if i == total {
			// add the current row to the keyboard
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, currentInlineRow)
		}

	}
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

func HandleMessage(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, openaiClient *openai.Client, db *sql.DB) {
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
	if message == "" {
		return "", errors.New("message is empty")
	}
	// check if we can find cached response
	log.Printf("Checking cache for response: language=%s, type=%s, word=%s\n", language, helpType, message)

	cachedResponse, err := storage.GetCachedResponseByWordLangAndType(db, language, helpType, message)
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
