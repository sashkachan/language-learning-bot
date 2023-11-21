package openai_api

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// type WordData struct {
// 	UsageExamples []string
// 	Translation   string
// 	Pronunciation string
// }

func GetGPTResponse(ctx context.Context, openaiClient *openai.Client, prompt string) (string, error) {
	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
