package config

import (
	template "text/template"
)

type Config struct {
	GptTemplateWordUsageExamples *template.Template
	GptTemplateWordTranslation   *template.Template
}

// NewConfig creates a new config
func NewConfig() *Config {
	config := &Config{
		GptTemplateWordUsageExamples: template.Must(template.ParseFiles("templates/word_usage_examples.txt")),
		GptTemplateWordTranslation:   template.Must(template.ParseFiles("templates/word_translation.txt")),
	}
	return config
}
