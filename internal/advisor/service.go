package advisor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/advisor"
	"loveguru/proto/common"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAdvisors(ctx context.Context, req *advisor.ListAdvisorsRequest) (*advisor.ListAdvisorsResponse, error) {
	advisors, err := s.repo.ListAdvisors(ctx, db.ListAdvisorsParams{
		LimitRows:  int32(req.Limit),
		OffsetRows: int32(req.Offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []*advisor.AdvisorWithRating
	for _, a := range advisors {
		resp = append(resp, &advisor.AdvisorWithRating{
			Advisor:       s.mapAdvisorFromRow(a),
			User:          s.mapUserFromRow(a),
			AverageRating: float64(a.AverageRating),
		})
	}

	return &advisor.ListAdvisorsResponse{Advisors: resp}, nil
}

func (s *Service) GetAdvisor(ctx context.Context, req *advisor.GetAdvisorRequest) (*advisor.GetAdvisorResponse, error) {
	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, err
	}

	a, err := s.repo.GetAdvisorByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &advisor.GetAdvisorResponse{
		Advisor: &advisor.AdvisorWithRating{
			Advisor: s.mapAdvisorFromGetRow(a),
			User:    s.mapUserFromGetRow(a),
		},
	}, nil
}

func (s *Service) ApplyAsAdvisor(ctx context.Context, req *advisor.ApplyAsAdvisorRequest) (*advisor.ApplyAsAdvisorResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	uid, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, err
	}

	a, err := s.repo.CreateAdvisor(ctx, db.CreateAdvisorParams{
		UserID:          uid,
		Bio:             sql.NullString{String: req.Bio, Valid: req.Bio != ""},
		ExperienceYears: sql.NullInt32{Int32: int32(req.ExperienceYears), Valid: req.ExperienceYears > 0},
		Languages:       req.Languages,
		Specializations: req.Specializations,
		HourlyRate:      sql.NullString{String: fmt.Sprintf("%.2f", req.HourlyRate), Valid: req.HourlyRate > 0},
	})
	if err != nil {
		return nil, err
	}

	return &advisor.ApplyAsAdvisorResponse{Advisor: s.mapAdvisor(a)}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, req *advisor.UpdateProfileRequest) (*advisor.UpdateProfileResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	uid, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, err
	}

	a, err := s.repo.UpdateAdvisor(ctx, db.UpdateAdvisorParams{
		ID:              uid,
		Bio:             sql.NullString{String: req.Bio, Valid: req.Bio != ""},
		ExperienceYears: sql.NullInt32{Int32: int32(req.ExperienceYears), Valid: req.ExperienceYears > 0},
		Languages:       req.Languages,
		Specializations: req.Specializations,
		HourlyRate:      sql.NullString{String: fmt.Sprintf("%.2f", req.HourlyRate), Valid: req.HourlyRate > 0},
		Status:          sql.NullString{String: req.Status.String(), Valid: req.Status != 0},
	})
	if err != nil {
		return nil, err
	}

	return &advisor.UpdateProfileResponse{Advisor: s.mapAdvisor(a)}, nil
}

func (s *Service) mapAdvisorFromRow(a db.ListAdvisorsRow) *common.Advisor {
	return &common.Advisor{
		Id:              a.ID.String(),
		UserId:          a.UserID.String(),
		Bio:             a.Bio.String,
		ExperienceYears: int32(a.ExperienceYears.Int32),
		Languages:       a.Languages,
		Specializations: a.Specializations,
		IsVerified:      a.IsVerified.Bool,
		HourlyRate:      parseFloat(a.HourlyRate.String),
		Status:          common.AdvisorStatus(common.AdvisorStatus_value[a.Status.String]),
		CreatedAt:       a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       a.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func (s *Service) mapUserFromRow(a db.ListAdvisorsRow) *common.User {
	return &common.User{
		Id:          a.UserID.String(),
		Email:       a.Email.String,
		Phone:       a.Phone.String,
		DisplayName: a.DisplayName,
		Role:        common.Role(common.Role_value[a.Role]),
		Gender:      common.Gender(common.Gender_value[a.Gender.String]),
		Dob:         a.Dob.Time.Format("2006-01-02"),
		CreatedAt:   a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   a.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		IsActive:    a.IsActive.Bool,
	}
}

func (s *Service) mapAdvisorFromGetRow(a db.GetAdvisorByIDRow) *common.Advisor {
	return &common.Advisor{
		Id:              a.ID.String(),
		UserId:          a.UserID.String(),
		Bio:             a.Bio.String,
		ExperienceYears: int32(a.ExperienceYears.Int32),
		Languages:       a.Languages,
		Specializations: a.Specializations,
		IsVerified:      a.IsVerified.Bool,
		HourlyRate:      parseFloat(a.HourlyRate.String),
		Status:          common.AdvisorStatus(common.AdvisorStatus_value[a.Status.String]),
		CreatedAt:       a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       a.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func (s *Service) mapUserFromGetRow(a db.GetAdvisorByIDRow) *common.User {
	return &common.User{
		Id:          a.UserID.String(),
		Email:       a.Email.String,
		Phone:       a.Phone.String,
		DisplayName: a.DisplayName,
		Role:        common.Role(common.Role_value[a.Role]),
		Gender:      common.Gender(common.Gender_value[a.Gender.String]),
		Dob:         a.Dob.Time.Format("2006-01-02"),
		CreatedAt:   a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   a.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		IsActive:    a.IsActive.Bool,
	}
}

func (s *Service) mapAdvisor(a db.Advisor) *common.Advisor {
	return &common.Advisor{
		Id:              a.ID.String(),
		UserId:          a.UserID.String(),
		Bio:             a.Bio.String,
		ExperienceYears: int32(a.ExperienceYears.Int32),
		Languages:       a.Languages,
		Specializations: a.Specializations,
		IsVerified:      a.IsVerified.Bool,
		HourlyRate:      parseFloat(a.HourlyRate.String),
		Status:          common.AdvisorStatus(common.AdvisorStatus_value[a.Status.String]),
		CreatedAt:       a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       a.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
