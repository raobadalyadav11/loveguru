package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/common"
	"loveguru/proto/user"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, req *user.GetProfileRequest) (*user.GetProfileResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	userID, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &user.GetProfileResponse{
		User: s.mapUser(u),
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, req *user.UpdateProfileRequest) (*user.UpdateProfileResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	userID, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	gender := sql.NullString{String: req.Gender.String(), Valid: true}
	dob := sql.NullTime{Time: parseTime(req.Dob), Valid: req.Dob != ""}

	u, err := s.repo.UpdateUser(ctx, db.UpdateUserParams{
		ID:          userID,
		DisplayName: req.DisplayName,
		Gender:      gender,
		Dob:         dob,
	})
	if err != nil {
		return nil, err
	}

	return &user.UpdateProfileResponse{
		User: s.mapUser(u),
	}, nil
}

func (s *Service) GetSessions(ctx context.Context, req *user.GetSessionsRequest) (*user.GetSessionsResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	userID, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	sessions, err := s.repo.GetUserSessions(ctx, db.GetUserSessionsParams{
		UserID: userID,
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
	})
	if err != nil {
		return nil, err
	}

	var sess []*common.Session
	for _, s := range sessions {
		sess = append(sess, &common.Session{
			Id:        s.ID.String(),
			UserId:    s.UserID.String(),
			AdvisorId: s.AdvisorID.UUID.String(),
			Type:      common.SessionType(common.SessionType_value[s.Type]),
			StartedAt: s.StartedAt.Time.Format("2006-01-02T15:04:05Z"),
			EndedAt:   s.EndedAt.Time.Format("2006-01-02T15:04:05Z"),
			Status:    common.SessionStatus(common.SessionStatus_value[s.Status.String]),
		})
	}

	return &user.GetSessionsResponse{
		Sessions: sess,
	}, nil
}

func (s *Service) mapUser(u db.User) *common.User {
	return &common.User{
		Id:          u.ID.String(),
		Email:       u.Email.String,
		Phone:       u.Phone.String,
		DisplayName: u.DisplayName,
		Role:        common.Role(common.Role_value[u.Role]),
		Gender:      common.Gender(common.Gender_value[u.Gender.String]),
		Dob:         u.Dob.Time.Format("2006-01-02"),
		CreatedAt:   u.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   u.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		IsActive:    u.IsActive.Bool,
	}
}

func parseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
