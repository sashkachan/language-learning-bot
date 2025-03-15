package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"language-learning-bot/pkg/config"
	openai_api "language-learning-bot/pkg/openai"
	storage "language-learning-bot/pkg/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

func HandleCommand(ctx context.Context, bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	// log the command to the console
	log.Printf("%d [%s] %s", message.From.ID, message.From.UserName, message.Text)
	response := ""
	switch message.Command() {
	case "healthz":
		response = "OK"
	case "start":
		if err := sendLanguageSelection(bot, message.Chat.ID); err != nil {
			log.Printf("Error sending language selection: %v\n", err)
			return err
		}
		response = ""
	case "speech_speed":
		if err := sendSpeechSpeedSelection(bot, message.Chat.ID); err != nil {
			log.Printf("Error sending speech speed selection: %v\n", err)
			return err
		}
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

	case "inflection":
		if err := handleInflectionCommand(bot, message, db, openaiClient); err != nil {
			log.Printf("Error handling inflection command: %v\n", err)
			return err
		}
		response = "I will respond with inflection (if applicable) for the provided word."
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

func handleInflectionCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	err := storage.UpdateUserHelpType(db, int(message.From.ID), "inflection")
	if err != nil {
		log.Printf("Error updating user help_type: %v\n", err)
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
	return nil
}

func handleTranslationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, openaiClient *openai.Client) error {
	err := storage.UpdateUserHelpType(db, int(message.From.ID), "translation")
	if err != nil {
		log.Printf("Error updating user help_type: %v\n", err)
		return err
	}
	return nil
}

func sendAudioMessage(openaiClient *openai.Client, db *sql.DB, firstLine string, userid int, bot *tgbotapi.BotAPI) error {
	userSpeechSpeed, err := storage.GetUserSpeechSpeed(db, userid)

	if err != nil {
		log.Println("Failed to get user speech speed: ", err)
		userSpeechSpeed = 1.0
	}

	openaiResponse, err := openai_api.GetTTSResponse(context.Background(), openaiClient, userSpeechSpeed, firstLine)

	if err != nil {
		log.Printf("Error getting TTS response: %v\n", err)
		return err
	}

	audio := tgbotapi.FileBytes{Name: fmt.Sprintf("%s.mp3", firstLine), Bytes: openaiResponse}
	audioMsg := tgbotapi.NewVoice(int64(userid), audio)
	_, err = bot.Send(audioMsg)
	if err != nil {
		log.Printf("Error sending audio message: %v\n", err)
		return err
	}
	return nil
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, openaiClient *openai.Client, callbackQuery *tgbotapi.CallbackQuery, db *sql.DB) {
	data := callbackQuery.Data
	if strings.HasPrefix(data, "language:") {
		language := strings.Split(data, ":")[1]
		updateLanguagePreference(bot, callbackQuery, db, language, 0)
	}

	if strings.HasPrefix(data, "pronunciation:") {
		// parse the number from the callback data into an int
		exampleNumber, err := strconv.Atoi(strings.Split(data, ":")[1])
		if err != nil {
			log.Printf("Error parsing example number: %v\n", err)
			return
		}
		log.Println("Pronounciation example: ", exampleNumber)

		msg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID,
			callbackQuery.Message.MessageID,
			fmt.Sprintf("You picked number %d. The pronunciation will be sent to you shortly."+
				"If it does not pop up in a few seconds, please choose /pronunciation from the menu and try again!", exampleNumber))
		_, err = bot.Send(msg)
		if err != nil {
			log.Printf("Error sending confirmation message: %v\n", err)
		}
		userId := int(callbackQuery.From.ID)

		// send the Nth example
		shouldReturn := sendLastRequestAudio(db, userId, exampleNumber, callbackQuery.Message.Text, openaiClient, bot)
		if shouldReturn {
			log.Printf("Error sending last request audio")
			return
		}
	}

	// set speech speed
	if strings.HasPrefix(data, "speech_speed:") {
		// parse the number from the callback data into an int
		speechSpeed, err := strconv.ParseFloat(strings.Split(data, ":")[1], 64)
		if err != nil {
			log.Printf("Error parsing speech speed: %v\n", err)
			return
		}
		log.Println("Speech speed: ", speechSpeed)

		speechSpeedValues := getSpeechSpeedValues()
		if speechSpeedText, ok := speechSpeedValues[speechSpeed]; ok {
			log.Printf("Setting speech speed to %.1f", speechSpeed)
			msg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID,
				callbackQuery.Message.MessageID,
				fmt.Sprintf("You picked %s speech speed. The speech speed will be applied to the next pronunciation.", speechSpeedText))
			_, err = bot.Send(msg)
			if err != nil {
				log.Printf("Error sending confirmation message: %v\n", err)
			}
			userId := int(callbackQuery.From.ID)

			// send the Nth example
			err = storage.UpdateUserSpeechSpeed(db, userId, speechSpeed)
			if err != nil {
				log.Printf("Error updating user speech speed: %v\n", err)
				return
			}
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
		log.Printf("Examples: %d\n", len(examples))

		if exampleNumber == 0 && len(examples) > 0 {
			// draw the inline keyboard with the examples
			err := sendExamplesSelection(bot, int64(userId), len(examples))
			if err != nil {
				log.Printf("Error sending examples selection: %v\n", err)
				return true
			}
		} else if len(examples) >= exampleNumber || len(examples) == 0 {
			pronunciationString := ""
			if len(examples) == 0 {
				pronunciationString = lastResponse
			} else {
				pronunciationString = examples[exampleNumber-1]
			}
			err := sendAudioMessage(openaiClient, db, pronunciationString, userId, bot)
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

			err := sendAudioMessage(openaiClient, db, firstLine, userId, bot)
			if err != nil {
				log.Printf("Error sending audio message: %v\n", err)
				return true
			}
		}
	} else {
		// examples count is 0?
	}
	return false
}

func sendLanguageSelection(bot *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Please choose a language you want help learning:")
	msg.ReplyMarkup = languageInlineKeyboard()
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending language selection: %v\n", err)
		return err
	}
	return nil
}

func sendSpeechSpeedSelection(bot *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Please choose a speech speed:")
	msg.ReplyMarkup = speechSpeedInlineKeyboard()
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending speech speed selection: %v\n", err)
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

func getSpeechSpeedValues() map[float64]string {
	speedValues := map[float64]string{
		0.5: "Slow",
		0.7: "Normal",
		1.0: "Fast",
	}

	keys := make([]float64, 0, len(speedValues))
	for k := range speedValues {
		keys = append(keys, k)
	}

	sort.Float64s(keys)

	sortedMap := make(map[float64]string)
	for _, k := range keys {
		sortedMap[k] = speedValues[k]
	}

	return sortedMap
}

// speechSpeedInlineKeyboard returns an inline keyboard with speech speed options
// The following options are available:
// - Slow - 0.5
// - Normal - 0.7
// - Fast - 1.0
// User is presented the text options
func speechSpeedInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	currentInlineRow := tgbotapi.NewInlineKeyboardRow()

	speechSpeedValues := getSpeechSpeedValues()
	for speechSpeed, speechSpeedText := range speechSpeedValues {
		currentInlineRow = append(currentInlineRow, tgbotapi.NewInlineKeyboardButtonData(speechSpeedText, fmt.Sprintf("speech_speed:%.1f", speechSpeed)))
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, currentInlineRow)
	return keyboard
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

func updateLanguagePreference(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *sql.DB, language string, speech_speed float64) {
	userID := int(callbackQuery.From.ID)
	err := storage.UpdateUserLanguage(db, userID, language)
	if err != nil {
		// Handle error
		log.Printf("Error updating language preference: %v\n", err)
		return
	}

	responseMsg := "Great, you picked %s. If you start typing words or phrases, I will send you a few examples with that word or a phrase. " +
		"If you type a whole sentence, then that sentence will be translated to %s. " +
		"You can also pick translation, where I will translate supplied phrase either from English to the language you picked, or the other way around. " +
		"Enjoy!"

	processedResponseMsg := fmt.Sprintf(responseMsg, language, language)
	// Send a confirmation message and remove the inline keyboard
	msg := tgbotapi.NewEditMessageText(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, processedResponseMsg)
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
	// send thinking message while the api is processing the request
	thinkMsgResponse, shouldReturn := sendThinkingMessage(message, bot)
	if shouldReturn {
		return
	}
	defer deleteThinkingMessage(message, thinkMsgResponse, bot)

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

func deleteThinkingMessage(message *tgbotapi.Message, thinkMsgResponse tgbotapi.Message, bot *tgbotapi.BotAPI) {
	deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, thinkMsgResponse.MessageID)
	response, err := bot.Request(deleteMsg)
	if err != nil {
		log.Printf("Error deleting thinking message: %v\n", err)
	}
	if string(response.Result) != "true" {
		log.Printf("response is not true from deleteThinkingMessage")
	}
}

func sendThinkingMessage(message *tgbotapi.Message, bot *tgbotapi.BotAPI) (tgbotapi.Message, bool) {
	thinkMsg := tgbotapi.NewMessage(message.Chat.ID, "Thinking...")
	thinkMsgResponse, err := bot.Send(thinkMsg)
	if err != nil {
		log.Printf("Error sending thinking message: %v\n", err)
		return tgbotapi.Message{}, true
	}
	return thinkMsgResponse, false
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
	case "inflection":
		gpt = gptConfig.GptTemplateInflection
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
		helpType = "translation"
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
