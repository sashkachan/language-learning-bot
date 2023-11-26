package openai_api

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

type GPTRequest struct {
	Prompt                 string
	WordOrPhrase           string
	ChatCompletionMessages []openai.ChatCompletionMessage
}

func GetGPTResponse(ctx context.Context, openaiClient *openai.Client, req GPTRequest) (string, error) {
	// Refactored implementation
	promptMessages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: req.Prompt},
	}

	promptAndMessages := append(promptMessages, req.ChatCompletionMessages...)
	promptAndMessages = append(promptAndMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.WordOrPhrase,
	})

	resp, err := openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo16K0613,
		Messages: promptAndMessages,
	})

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
