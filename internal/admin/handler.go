package admin

import (
	"context"
	"loveguru/proto/admin"
)

type Handler struct {
	admin.UnimplementedAdminServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetPendingAdvisors(ctx context.Context, req *admin.GetPendingAdvisorsRequest) (*admin.GetPendingAdvisorsResponse, error) {
	return h.service.GetPendingAdvisors(ctx, req)
}

func (h *Handler) ApproveAdvisor(ctx context.Context, req *admin.ApproveAdvisorRequest) (*admin.ApproveAdvisorResponse, error) {
	return h.service.ApproveAdvisor(ctx, req)
}

func (h *Handler) GetFlags(ctx context.Context, req *admin.GetFlagsRequest) (*admin.GetFlagsResponse, error) {
	return h.service.GetFlags(ctx, req)
}

func (h *Handler) BlockUser(ctx context.Context, req *admin.BlockUserRequest) (*admin.BlockUserResponse, error) {
	return h.service.BlockUser(ctx, req)
}
