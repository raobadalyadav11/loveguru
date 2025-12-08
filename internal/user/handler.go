package user

import (
	"context"
	"loveguru/proto/user"
)

type Handler struct {
	user.UnimplementedUserServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetProfile(ctx context.Context, req *user.GetProfileRequest) (*user.GetProfileResponse, error) {
	return h.service.GetProfile(ctx, req)
}

func (h *Handler) UpdateProfile(ctx context.Context, req *user.UpdateProfileRequest) (*user.UpdateProfileResponse, error) {
	return h.service.UpdateProfile(ctx, req)
}

func (h *Handler) GetSessions(ctx context.Context, req *user.GetSessionsRequest) (*user.GetSessionsResponse, error) {
	return h.service.GetSessions(ctx, req)
}
