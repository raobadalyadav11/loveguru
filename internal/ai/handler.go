package ai

import (
	"context"
	"loveguru/proto/ai"
)

type Handler struct {
	ai.UnimplementedAIServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	return h.service.Chat(ctx, req)
}

func (h *Handler) ChatStream(stream ai.AIService_ChatStreamServer) error {
	return h.service.ChatStream(nil, stream)
}
