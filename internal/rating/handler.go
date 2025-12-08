package rating

import (
	"context"
	"loveguru/proto/rating"
)

type Handler struct {
	rating.UnimplementedRatingServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateRating(ctx context.Context, req *rating.CreateRatingRequest) (*rating.CreateRatingResponse, error) {
	return h.service.CreateRating(ctx, req)
}

func (h *Handler) GetAdvisorRatings(ctx context.Context, req *rating.GetAdvisorRatingsRequest) (*rating.GetAdvisorRatingsResponse, error) {
	return h.service.GetAdvisorRatings(ctx, req)
}
