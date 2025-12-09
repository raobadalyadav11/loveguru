package admin

import (
	"context"
	"errors"
	"strconv"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/admin"
	"loveguru/proto/common"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetPendingAdvisors(ctx context.Context, req *admin.GetPendingAdvisorsRequest) (*admin.GetPendingAdvisorsResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok || userInfo.Role != "ADMIN" {
		return nil, errors.New("unauthorized")
	}

	advisors, err := s.repo.GetPendingAdvisors(ctx, db.GetPendingAdvisorsParams{
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []*common.Advisor
	for _, a := range advisors {
		resp = append(resp, &common.Advisor{
			Id:              a.ID.String(),
			UserId:          a.UserID.String(),
			Bio:             a.Bio.String,
			ExperienceYears: int32(a.ExperienceYears.Int32),
			Languages:       a.Languages,
			Specializations: a.Specializations,
			IsVerified:      a.IsVerified.Bool,
			HourlyRate:      func() float64 { f, _ := strconv.ParseFloat(a.HourlyRate.String, 64); return f }(),
			Status:          common.AdvisorStatus(common.AdvisorStatus_value[a.Status.String]),
			CreatedAt:       a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       a.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &admin.GetPendingAdvisorsResponse{Advisors: resp}, nil
}

func (s *Service) ApproveAdvisor(ctx context.Context, req *admin.ApproveAdvisorRequest) (*admin.ApproveAdvisorResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok || userInfo.Role != "ADMIN" {
		return nil, errors.New("unauthorized")
	}

	aid, err := uuid.Parse(req.AdvisorId)
	if err != nil {
		return nil, err
	}

	err = s.repo.ApproveAdvisor(ctx, aid)
	if err != nil {
		return nil, err
	}

	return &admin.ApproveAdvisorResponse{Success: true}, nil
}

func (s *Service) GetFlags(ctx context.Context, req *admin.GetFlagsRequest) (*admin.GetFlagsResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok || userInfo.Role != "ADMIN" {
		return nil, errors.New("unauthorized")
	}

	flags, err := s.repo.GetFlags(ctx, db.GetFlagsParams{
		Limit:  int32(req.Limit),
		Offset: int32(req.Offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []*admin.AdminFlag
	for _, f := range flags {
		resp = append(resp, &admin.AdminFlag{
			Id:                f.ID.String(),
			ReportedBy:        f.ReportedBy.String(),
			ReportedUserId:    f.ReportedUserID.UUID.String(),
			ReportedAdvisorId: f.ReportedAdvisorID.UUID.String(),
			Reason:            f.Reason,
			SessionId:         f.SessionID.UUID.String(),
			CreatedAt:         f.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			Status:            f.Status.String,
		})
	}

	return &admin.GetFlagsResponse{Flags: resp}, nil
}

func (s *Service) BlockUser(ctx context.Context, req *admin.BlockUserRequest) (*admin.BlockUserResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok || userInfo.Role != "ADMIN" {
		return nil, errors.New("unauthorized")
	}

	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, err
	}

	err = s.repo.BlockUser(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &admin.BlockUserResponse{Success: true}, nil
}

// TODO: Implement specialization management once database queries are available
/*
func (s *Service) GetAllSpecializations(ctx context.Context) ([]Specialization, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) GetActiveSpecializationsByCategory(ctx context.Context, category string) ([]Specialization, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) CreateSpecialization(ctx context.Context, name, description, category string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *Service) UpdateSpecialization(ctx context.Context, specID, name, description, category string, isActive bool) error {
	return errors.New("not implemented")
}

func (s *Service) DeleteSpecialization(ctx context.Context, specID string) error {
	return errors.New("not implemented")
}

func (s *Service) GetUserSpecializations(ctx context.Context, userID string) ([]Specialization, error) {
	return nil, errors.New("not implemented")
}
*/

type Specialization struct {
	ID          string
	Name        string
	Description string
	Category    string
	IsActive    bool
}
