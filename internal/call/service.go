package call

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/call"
	"loveguru/proto/common"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
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

	// TODO: Integrate with VoIP provider to get token and room
	callToken := "dummy_token"
	roomID := "dummy_room"

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

	// TODO: Get call duration from VoIP provider
	duration := 300 // dummy

	_, err = s.repo.InsertCallLog(ctx, db.InsertCallLogParams{
		SessionID:       sid,
		ExternalCallID:  sql.NullString{String: "dummy", Valid: true},
		StartedAt:       sql.NullTime{Valid: false},
		EndedAt:         sql.NullTime{Time: time.Now(), Valid: true},
		DurationSeconds: sql.NullInt32{Int32: int32(duration), Valid: true},
		Status:          sql.NullString{String: "ENDED", Valid: true},
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
