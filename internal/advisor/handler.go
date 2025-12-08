package advisor

import (
	"context"
	"loveguru/proto/advisor"
)

type Handler struct {
	advisor.UnimplementedAdvisorServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListAdvisors(ctx context.Context, req *advisor.ListAdvisorsRequest) (*advisor.ListAdvisorsResponse, error) {
	return h.service.ListAdvisors(ctx, req)
}

func (h *Handler) GetAdvisor(ctx context.Context, req *advisor.GetAdvisorRequest) (*advisor.GetAdvisorResponse, error) {
	return h.service.GetAdvisor(ctx, req)
}

func (h *Handler) ApplyAsAdvisor(ctx context.Context, req *advisor.ApplyAsAdvisorRequest) (*advisor.ApplyAsAdvisorResponse, error) {
	return h.service.ApplyAsAdvisor(ctx, req)
}

func (h *Handler) UpdateProfile(ctx context.Context, req *advisor.UpdateProfileRequest) (*advisor.UpdateProfileResponse, error) {
	return h.service.UpdateProfile(ctx, req)
}
