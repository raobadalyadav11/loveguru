package rating

import (
	"context"
	"database/sql"
	"errors"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/common"
	"loveguru/proto/rating"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateRating(ctx context.Context, req *rating.CreateRatingRequest) (*rating.CreateRatingResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	uid, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, err
	}

	sid, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, err
	}

	// Get session to find advisor
	session, err := s.repo.GetSessionByID(ctx, sid)
	if err != nil {
		return nil, err
	}

	if !session.AdvisorID.Valid {
		return nil, errors.New("session has no advisor")
	}

	r, err := s.repo.CreateRating(ctx, db.CreateRatingParams{
		SessionID:  sid,
		UserID:     uid,
		AdvisorID:  session.AdvisorID.UUID,
		Rating:     int32(req.Rating),
		ReviewText: sql.NullString{String: req.ReviewText, Valid: req.ReviewText != ""},
	})
	if err != nil {
		return nil, err
	}

	return &rating.CreateRatingResponse{
		Rating: &common.Rating{
			Id:         r.ID.String(),
			SessionId:  r.SessionID.String(),
			UserId:     r.UserID.String(),
			AdvisorId:  r.AdvisorID.String(),
			Rating:     r.Rating,
			ReviewText: r.ReviewText.String,
			CreatedAt:  r.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

func (s *Service) GetAdvisorRatings(ctx context.Context, req *rating.GetAdvisorRatingsRequest) (*rating.GetAdvisorRatingsResponse, error) {
	aid, err := uuid.Parse(req.AdvisorId)
	if err != nil {
		return nil, err
	}

	ratings, err := s.repo.GetAdvisorRatings(ctx, db.GetAdvisorRatingsParams{
		AdvisorID: aid,
		Limit:     int32(req.Limit),
		Offset:    int32(req.Offset),
	})
	if err != nil {
		return nil, err
	}

	var rs []*common.Rating
	for _, r := range ratings {
		rs = append(rs, &common.Rating{
			Id:         r.ID.String(),
			SessionId:  r.SessionID.String(),
			UserId:     r.UserID.String(),
			AdvisorId:  r.AdvisorID.String(),
			Rating:     r.Rating,
			ReviewText: r.ReviewText.String,
			CreatedAt:  r.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &rating.GetAdvisorRatingsResponse{Ratings: rs}, nil
}
