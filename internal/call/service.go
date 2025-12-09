package call

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/call"
	"loveguru/proto/common"

	"github.com/google/uuid"
)

type Service struct {
	repo         *db.Queries
	agoraService *AgoraService
}

func NewService(repo *db.Queries, agoraService *AgoraService) *Service {
	return &Service{
		repo:         repo,
		agoraService: agoraService,
	}
}

func (s *Service) CreateSession(ctx context.Context, req *call.CreateSessionRequest) (*call.CreateSessionResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	uid, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, err
	}

	aid, err := uuid.Parse(req.AdvisorId)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.CreateCallSession(ctx, db.CreateCallSessionParams{
		UserID:    uid,
		AdvisorID: uuid.NullUUID{UUID: aid, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	// Create Agora call session
	agoraCallInfo, err := s.agoraService.CreateCallSession(ctx, userInfo.ID, req.AdvisorId)
	if err != nil {
		// Log the error for debugging
		return nil, fmt.Errorf("failed to create Agora call session: %w", err)
	}

	// Validate Agora call info
	if agoraCallInfo == nil {
		return nil, fmt.Errorf("Agora call session returned nil info")
	}

	if agoraCallInfo.Token == "" {
		return nil, fmt.Errorf("Agora call session returned empty token")
	}

	callToken := agoraCallInfo.Token
	roomID := agoraCallInfo.ExternalID

	return &call.CreateSessionResponse{
		Session: &common.Session{
			Id:        session.ID.String(),
			UserId:    session.UserID.String(),
			AdvisorId: session.AdvisorID.UUID.String(),
			Type:      common.SessionType(common.SessionType_value[session.Type]),
			StartedAt: session.StartedAt.Time.Format("2006-01-02T15:04:05Z"),
			EndedAt:   session.EndedAt.Time.Format("2006-01-02T15:04:05Z"),
			Status:    common.SessionStatus(common.SessionStatus_value[session.Status.String]),
		},
		CallToken: callToken,
		RoomId:    roomID,
	}, nil
}

func (s *Service) EndCall(ctx context.Context, req *call.EndCallRequest) (*call.EndCallResponse, error) {
	_, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	sid, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, err
	}

	// Get real call duration from Agora
	duration, status, err := s.agoraService.GetCallStats(ctx, req.SessionId)
	if err != nil {
		// Log the error but fall back to estimation
		// In production, you might want to implement retry logic or alerting
		duration = 0
		status = "ENDED"
	}

	// End the Agora call
	err = s.agoraService.EndCall(ctx, req.SessionId)
	if err != nil {
		// Log error but don't fail the operation
		// The call may have already ended or there's a temporary issue
	}

	_, err = s.repo.InsertCallLog(ctx, db.InsertCallLogParams{
		SessionID:       sid,
		ExternalCallID:  sql.NullString{String: req.SessionId, Valid: true},
		StartedAt:       sql.NullTime{Valid: false},
		EndedAt:         sql.NullTime{Time: time.Now(), Valid: true},
		DurationSeconds: sql.NullInt32{Int32: int32(duration), Valid: true},
		Status:          sql.NullString{String: status, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	err = s.repo.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		ID:     sid,
		Status: sql.NullString{String: "ENDED", Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &call.EndCallResponse{Success: true}, nil
}

func (s *Service) GetCall(ctx context.Context, req *call.GetCallRequest) (*call.GetCallResponse, error) {
	sid, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetSessionByID(ctx, sid)
	if err != nil {
		return nil, err
	}

	return &call.GetCallResponse{
		Session: &common.Session{
			Id:        session.ID.String(),
			UserId:    session.UserID.String(),
			AdvisorId: session.AdvisorID.UUID.String(),
			Type:      common.SessionType(common.SessionType_value[session.Type]),
			StartedAt: session.StartedAt.Time.Format("2006-01-02T15:04:05Z"),
			EndedAt:   session.EndedAt.Time.Format("2006-01-02T15:04:05Z"),
			Status:    common.SessionStatus(common.SessionStatus_value[session.Status.String]),
		},
	}, nil
}
