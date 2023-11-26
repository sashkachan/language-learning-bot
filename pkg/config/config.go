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

					promptTuning := GptPromptTuning{
						Language: language,
						HelpType: helpType,
						Messages: []openai.ChatCompletionMessage{
							{
								Role:    openai.ChatMessageRoleUser,
								Content: string(content),
							},
						},
					}

					if _, ok := promptTunings[language]; !ok {
						promptTunings[language] = make(map[string]GptPromptTuning)
					}
					promptTunings[language][helpType] = promptTuning
				}
			}
		}
	}

	return promptTunings, nil
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
