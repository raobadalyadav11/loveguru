package ai

import (
	"context"
	"errors"
	"fmt"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/ai"

	"github.com/google/uuid"
)

type Service struct {
	repo   *db.Queries
	openai *OpenAIClient
}

func NewService(repo *db.Queries, apiKey, baseURL string) *Service {
	return &Service{
		repo:   repo,
		openai: NewOpenAIClient(apiKey, baseURL),
	}
}

func (s *Service) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	uid, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, err
	}

	// Use context parameter for session context
	var contextMessages []string
	if req.Context != "" {
		// Parse context for session ID or use context directly as previous messages
		contextMessages = []string{req.Context}
	}

	// Call OpenAI API
	response, err := s.openai.Chat(ctx, req.Message, contextMessages)
	if err != nil {
		return nil, err
	}

	// Store interaction
	_, err = s.repo.InsertAIInteraction(ctx, db.InsertAIInteractionParams{
		UserID:   uid,
		Prompt:   req.Message,
		Response: response,
	})
	if err != nil {
		return nil, err
	}

	return &ai.ChatResponse{Response: response}, nil
}

func (s *Service) ChatStream(req *ai.ChatMessage, stream ai.AIService_ChatStreamServer) error {
	// For simplicity, just echo
	resp := &ai.ChatMessage{
		Message: req.Message,
		Context: req.Context,
	}
	return stream.Send(resp)
}

func (s *Service) callAI(prompt string) (string, error) {
	// Dummy implementation
	// In real implementation, call OpenAI or other LLM API
	return fmt.Sprintf("AI response to: %s", prompt), nil
}
