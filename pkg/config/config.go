package config

import (
	"os"
	"path/filepath"
	"strings"
	template "text/template"

	"github.com/sashabaranov/go-openai"
)

type GptPromptTuningByLanguageAndHelpType map[string]map[string]GptPromptTuning

type GptPromptTuning struct {
	Language string
	HelpType string
	Messages []openai.ChatCompletionMessage
}

type GptRequestType struct {
	HelpType       string
	PromptTemplate *template.Template
	Messages       []openai.ChatCompletionMessage
}

type Config struct {
	GptTemplateWordUsageExamples *GptRequestType
	GptTemplateWordTranslation   *GptRequestType
	GptPromptTunings             GptPromptTuningByLanguageAndHelpType
}

func NewGptPromptTuningFromTextFiles() (GptPromptTuningByLanguageAndHelpType, error) {
	promptTunings := make(GptPromptTuningByLanguageAndHelpType)

	// Read all files in templates/examples and templates/translation directories
	helpTypeDirectory, err := os.ReadDir("templates/")
	if err != nil {
		return nil, err
	}

	for _, file := range helpTypeDirectory {
		if file.IsDir() {
			helpType := file.Name()
			helpTypeDirectory, err := os.ReadDir(filepath.Join("templates", helpType))
			if err != nil {
				return nil, err
			}
			for _, file := range helpTypeDirectory {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
					language := strings.TrimSuffix(file.Name(), ".txt")
					filePath := filepath.Join("templates", helpType, file.Name())

					content, err := os.ReadFile(filePath)
					if err != nil {
						return nil, err
					}
					var chatCompletionMessages []openai.ChatCompletionMessage
					chatCompletionMessages = getChatCompletionMessages(content, chatCompletionMessages)
					promptTuning := GptPromptTuning{
						Language: language,
						HelpType: helpType,
						Messages: chatCompletionMessages,
					}

					if _, ok := promptTunings[language]; !ok {
						promptTunings[language] = make(map[string]GptPromptTuning)
					}
					promptTunings[language][helpType] = promptTuning
					chatCompletionMessages = nil
				}
			}
		}
	}

	return promptTunings, nil
}

func getChatCompletionMessages(content []byte, chatCompletionMessages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	for _, line := range strings.Split(string(content), "\n") {
		roleandcontent := strings.Split(line, ":")
		if len(roleandcontent) != 2 {
			continue
		}
		role := strings.TrimSpace(roleandcontent[0])
		content := strings.TrimSpace(roleandcontent[1])
		chatCompletionMessages = append(chatCompletionMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: strings.Replace(content, `\n`, "\n", -1),
		})
	}
	return chatCompletionMessages
}

// NewConfig creates a new config
func NewConfig() *Config {
	gptPromptTunings, err := NewGptPromptTuningFromTextFiles()
	if err != nil {
		panic(err)
	}

	config := &Config{
		GptPromptTunings: gptPromptTunings,
		GptTemplateWordUsageExamples: &GptRequestType{
			HelpType:       "examples",
			PromptTemplate: template.Must(template.ParseFiles("templates/examples.txt")),
		},

		GptTemplateWordTranslation: &GptRequestType{
			HelpType:       "translation",
			PromptTemplate: template.Must(template.ParseFiles("templates/translation.txt")),
		},
	}
	return config
}
