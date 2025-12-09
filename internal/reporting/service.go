package reporting

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

type Report struct {
	ID                string
	ReportedBy        string
	ReportedUserID    sql.NullString
	ReportedAdvisorID sql.NullString
	SessionID         sql.NullString
	Reason            string
	Status            string
	CreatedAt         time.Time
}

type ReportRequest struct {
	ReportedUserID    *string
	ReportedAdvisorID *string
	SessionID         *string
	Reason            string
	AdditionalDetails string
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
}

func (s *Service) ReportUser(ctx context.Context, req *ReportRequest) error {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return errors.New("unauthenticated")
	}

	reporterID, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return err
	}

	var reportedUserID uuid.NullUUID
	var reportedAdvisorID uuid.NullUUID
	var sessionID uuid.NullUUID

	if req.ReportedUserID != nil {
		if id, err := uuid.Parse(*req.ReportedUserID); err == nil {
			reportedUserID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	if req.ReportedAdvisorID != nil {
		if id, err := uuid.Parse(*req.ReportedAdvisorID); err == nil {
			reportedAdvisorID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	if req.SessionID != nil {
		if id, err := uuid.Parse(*req.SessionID); err == nil {
			sessionID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	_, err = s.repo.CreateAdminFlag(ctx, db.CreateAdminFlagParams{
		ReportedBy:        reporterID,
		ReportedUserID:    reportedUserID,
		ReportedAdvisorID: reportedAdvisorID,
		SessionID:         sessionID,
		Reason:            req.Reason,
	})

	return err
}

func (s *Service) BlockUser(ctx context.Context, targetUserID string) error {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok || userInfo.Role != "ADMIN" {
		return errors.New("unauthorized")
	}

	targetID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}

	return s.repo.BlockUser(ctx, targetID)
}

func (s *Service) GetUserReports(ctx context.Context, userID string) ([]Report, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	reports, err := s.repo.GetUserReports(ctx, uuid.NullUUID{UUID: uid, Valid: true})
	if err != nil {
		return nil, err
	}

	var reportList []Report
	for _, report := range reports {
		reportList = append(reportList, Report{
			ID:                report.ID.String(),
			ReportedBy:        report.ReportedBy.String(),
			ReportedUserID:    sql.NullString{String: report.ReportedUserID.UUID.String(), Valid: report.ReportedUserID.Valid},
			ReportedAdvisorID: sql.NullString{String: report.ReportedAdvisorID.UUID.String(), Valid: report.ReportedAdvisorID.Valid},
			SessionID:         sql.NullString{String: report.SessionID.UUID.String(), Valid: report.SessionID.Valid},
			Reason:            report.Reason,
			Status:            report.Status.String,
			CreatedAt:         report.CreatedAt.Time,
		})
	}

	return reportList, nil
}

func (s *Service) GetReportsByStatus(ctx context.Context, status string) ([]Report, error) {
	reports, err := s.repo.GetReportsByStatus(ctx, sql.NullString{String: status, Valid: true})
	if err != nil {
		return nil, err
	}

	var reportList []Report
	for _, report := range reports {
		reportList = append(reportList, Report{
			ID:                report.ID.String(),
			ReportedBy:        report.ReportedBy.String(),
			ReportedUserID:    sql.NullString{String: report.ReportedUserID.UUID.String(), Valid: report.ReportedUserID.Valid},
			ReportedAdvisorID: sql.NullString{String: report.ReportedAdvisorID.UUID.String(), Valid: report.ReportedAdvisorID.Valid},
			SessionID:         sql.NullString{String: report.SessionID.UUID.String(), Valid: report.SessionID.Valid},
			Reason:            report.Reason,
			Status:            report.Status.String,
			CreatedAt:         report.CreatedAt.Time,
		})
	}

	return reportList, nil
}

func (s *Service) ResolveReport(ctx context.Context, reportID, resolution, adminID string) error {
	reportUUID, err := uuid.Parse(reportID)
	if err != nil {
		return err
	}

	return s.repo.UpdateAdminFlagStatus(ctx, db.UpdateAdminFlagStatusParams{
		ID:     reportUUID,
		Status: sql.NullString{String: resolution, Valid: true},
	})
}

func (s *Service) IsUserBlocked(ctx context.Context, userID string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}

	user, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		return false, err
	}

	return !user.IsActive.Bool, nil
}

func (s *Service) GetAbuseStats(ctx context.Context) (*AbuseStats, error) {
	totalReports, err := s.repo.CountTotalReports(ctx)
	if err != nil {
		return nil, err
	}

	pendingReports, err := s.repo.CountPendingReports(ctx)
	if err != nil {
		return nil, err
	}

	resolvedReports, err := s.repo.CountResolvedReports(ctx)
	if err != nil {
		return nil, err
	}

	recentReports, err := s.repo.GetRecentAdminFlags(ctx)
	if err != nil {
		return nil, err
	}

	return &AbuseStats{
		TotalReports:    int32(totalReports),
		PendingReports:  int32(pendingReports),
		ResolvedReports: int32(resolvedReports),
		RecentReports:   int32(len(recentReports)),
		ResolutionRate:  float64(resolvedReports) / float64(totalReports) * 100,
	}, nil
}

type AbuseStats struct {
	TotalReports    int32
	PendingReports  int32
	ResolvedReports int32
	RecentReports   int32
	ResolutionRate  float64
}
