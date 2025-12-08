package call

import (
	"context"
	"loveguru/proto/call"
)

type Handler struct {
	call.UnimplementedCallServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateSession(ctx context.Context, req *call.CreateSessionRequest) (*call.CreateSessionResponse, error) {
	return h.service.CreateSession(ctx, req)
}

func (h *Handler) EndCall(ctx context.Context, req *call.EndCallRequest) (*call.EndCallResponse, error) {
	return h.service.EndCall(ctx, req)
}

func (h *Handler) GetCall(ctx context.Context, req *call.GetCallRequest) (*call.GetCallResponse, error) {
	return h.service.GetCall(ctx, req)
}
