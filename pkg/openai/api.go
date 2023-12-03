package openai_api

import (
	"context"
	"io"

	"log"

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
		{Role: openai.ChatMessageRoleSystem, Content: req.Prompt},
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

func GetTTSResponse(ctx context.Context, openaiClient *openai.Client, speechSpeed float64, req string) ([]byte, error) {
	request := openai.CreateSpeechRequest{
		Model: openai.TTSModel1,
		Input: req,
		Voice: openai.VoiceNova,
		Speed: speechSpeed,
	}
	log.Printf("GetTTSResponse request: speed=%.1f req=%s", speechSpeed, req)
	response, err := openaiClient.CreateSpeech(ctx, request)
	if err != nil {
		log.Println("error when requesting whisperapi")
		return nil, err
	}
	defer func(io.ReadCloser) {
		response.Close()
	}(response)

	body, err := io.ReadAll(response)
	if err != nil {
		log.Println("error when reading response body")
		return nil, err
	}

	response.Close()

	return body, nil
}
