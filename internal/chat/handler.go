package chat

import (
	"context"
	"loveguru/proto/chat"
)

type Handler struct {
	chat.UnimplementedChatServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateSession(ctx context.Context, req *chat.CreateSessionRequest) (*chat.CreateSessionResponse, error) {
	return h.service.CreateSession(ctx, req)
}

func (h *Handler) GetMessages(ctx context.Context, req *chat.GetMessagesRequest) (*chat.GetMessagesResponse, error) {
	return h.service.GetMessages(ctx, req)
}

func (h *Handler) ChatStream(stream chat.ChatService_ChatStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		// For now, just echo the message back
		resp := &chat.ChatMessageResponse{
			Message: &chat.ChatMessage{
				SessionId: req.SessionId,
				Content:   req.Content,
			},
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
