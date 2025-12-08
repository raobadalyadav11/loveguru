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
	repo    *db.Queries
	apiKey  string
	baseURL string
}

func NewService(repo *db.Queries, apiKey, baseURL string) *Service {
	return &Service{
		repo:    repo,
		apiKey:  apiKey,
		baseURL: baseURL,
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

	// Call external AI API
	response, err := s.callAI(req.Message + " " + req.Context)
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
