package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float32         `json:"temperature"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}

type OpenAIClient struct {
	apiKey    string
	baseURL   string
	model     string
	maxTokens int
	client    *http.Client
}

func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
	return NewOpenAIClientWithConfig(apiKey, baseURL, "gpt-3.5-turbo", 500)
}

func NewOpenAIClientWithConfig(apiKey, baseURL, model string, maxTokens int) *OpenAIClient {
	return &OpenAIClient{
		apiKey:    apiKey,
		baseURL:   baseURL,
		model:     model,
		maxTokens: maxTokens,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *OpenAIClient) Chat(ctx context.Context, prompt string, context []string) (string, error) {
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: "You are a professional love advice counselor. Provide helpful, empathetic, and constructive advice about relationships, dating, breakups, and love problems. Be supportive but honest. Keep responses concise and actionable.",
		},
	}

	// Add context messages if provided
	for _, msg := range context {
		messages = append(messages, OpenAIMessage{
			Role:    "user",
			Content: msg,
		})
	}

	// Add the current prompt
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: prompt,
	})

	req := OpenAIRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	var aiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", err
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return aiResp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) ChatWithContext(ctx context.Context, sessionID, prompt string, previousMessages []string) (string, error) {
	// Format previous messages as context
	var context []string
	for i := 0; i < len(previousMessages); i += 2 {
		if i+1 < len(previousMessages) {
			context = append(context, fmt.Sprintf("User: %s\nAssistant: %s", previousMessages[i], previousMessages[i+1]))
		} else {
			context = append(context, fmt.Sprintf("User: %s", previousMessages[i]))
		}
	}

	return c.Chat(ctx, prompt, context)
}
