package gpt

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

type Role string

const baseUri = "https://api.openai.com/v1/chat/completions"
const model = "gpt-4"
const (
	SystemRole    Role = "system"
	AssistantRole Role = "assistant"
	UserRole      Role = "user"
)

type ChatMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequestDTO struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatCompletionResponseDTO struct {
	Choice []ChatCompletionChoice `json:"choices"`
}

type ChatCompletionChoice struct {
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

func PostChatCompletion(chatMessages []ChatMessage) (*ChatCompletionResponseDTO, error) {
	client := &http.Client{}
	requestBody, err := json.Marshal(
		ChatCompletionRequestDTO{
			Model:    model,
			Messages: chatMessages,
		},
	)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseUri, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("CHAT_GPT_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("ChatGPT is not available")
	}

	var responseDTO ChatCompletionResponseDTO
	err = json.NewDecoder(resp.Body).Decode(&responseDTO)
	if err != nil {
		return nil, err
	}

	return &responseDTO, nil
}
