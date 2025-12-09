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

// TODO: Implement these methods once protobuf types are generated
/*
func (h *Handler) CreateAnonymousProfile(ctx context.Context, req *user.CreateAnonymousProfileRequest) (*user.CreateAnonymousProfileResponse, error) {
	return h.service.CreateAnonymousProfile(ctx, req)
}

func (h *Handler) ConvertAnonymousToFull(ctx context.Context, req *user.ConvertAnonymousToFullRequest) (*user.ConvertAnonymousToFullResponse, error) {
	return h.service.ConvertAnonymousToFull(ctx, req)
}

func (h *Handler) ForgotPassword(ctx context.Context, req *user.ForgotPasswordRequest) (*user.ForgotPasswordResponse, error) {
	return h.service.ForgotPassword(ctx, req)
}

func (h *Handler) ResetPassword(ctx context.Context, req *user.ResetPasswordRequest) (*user.ResetPasswordResponse, error) {
	return h.service.ResetPassword(ctx, req)
}
*/
