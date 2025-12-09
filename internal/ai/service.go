package ai

import (
	"context"
	"errors"

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
	return NewServiceWithConfig(repo, apiKey, baseURL, "gpt-3.5-turbo", 500)
}

func NewServiceWithConfig(repo *db.Queries, apiKey, baseURL, model string, maxTokens int) *Service {
	return &Service{
		repo:   repo,
		openai: NewOpenAIClientWithConfig(apiKey, baseURL, model, maxTokens),
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

func (s *Service) ChatStream(stream ai.AIService_ChatStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		// Process the message with OpenAI
		response, err := s.openai.Chat(stream.Context(), req.Message, []string{req.Context})
		if err != nil {
			// Send error message back to client
			errorResp := &ai.ChatMessage{
				Message: "Sorry, I encountered an error processing your request. Please try again.",
				Context: req.Context,
			}
			if err := stream.Send(errorResp); err != nil {
				return err
			}
			continue
		}

		// Send OpenAI response back to client
		resp := &ai.ChatMessage{
			Message: response,
			Context: req.Context,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
