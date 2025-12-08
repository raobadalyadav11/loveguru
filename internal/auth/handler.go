package auth

import (
	"context"

	"loveguru/proto/auth"
)

type Handler struct {
	auth.UnimplementedAuthServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	return h.service.Register(ctx, req)
}

func (h *Handler) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	return h.service.Login(ctx, req)
}

func (h *Handler) Refresh(ctx context.Context, req *auth.RefreshRequest) (*auth.RefreshResponse, error) {
	return h.service.Refresh(ctx, req)
}

func (h *Handler) Logout(ctx context.Context, req *auth.LogoutRequest) (*auth.LogoutResponse, error) {
	return h.service.Logout(ctx, req)
}
