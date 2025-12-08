package chat

import (
	"context"
	"database/sql"
	"errors"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/proto/chat"
	"loveguru/proto/common"

	"github.com/google/uuid"
)

type Service struct {
	repo *db.Queries
}

func NewService(repo *db.Queries) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateSession(ctx context.Context, req *chat.CreateSessionRequest) (*chat.CreateSessionResponse, error) {
	userInfo, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	uid, err := uuid.Parse(userInfo.ID)
	if err != nil {
		return nil, err
	}

	var advisorID uuid.NullUUID
	if req.AdvisorId != "" {
		aid, err := uuid.Parse(req.AdvisorId)
		if err != nil {
			return nil, err
		}
		advisorID = uuid.NullUUID{UUID: aid, Valid: true}
	}

	session, err := s.repo.CreateSession(ctx, db.CreateSessionParams{
		UserID:    uid,
		AdvisorID: advisorID,
		Type:      req.Type.String(),
	})
	if err != nil {
		return nil, err
	}

	return &chat.CreateSessionResponse{
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

func (s *Service) GetMessages(ctx context.Context, req *chat.GetMessagesRequest) (*chat.GetMessagesResponse, error) {
	_, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("unauthenticated")
	}

	sid, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, err
	}

	messages, err := s.repo.GetMessages(ctx, db.GetMessagesParams{
		SessionID: sid,
		Limit:     int32(req.Limit),
		Offset:    int32(req.Offset),
	})
	if err != nil {
		return nil, err
	}

	var msgs []*common.ChatMessage
	for _, m := range messages {
		msgs = append(msgs, &common.ChatMessage{
			Id:         m.ID.String(),
			SessionId:  m.SessionID.String(),
			SenderType: m.SenderType,
			SenderId:   m.SenderID.String(),
			Content:    m.Content,
			CreatedAt:  m.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			IsRead:     m.IsRead.Bool,
		})
	}

	return &chat.GetMessagesResponse{Messages: msgs}, nil
}

func (s *Service) InsertMessage(ctx context.Context, sessionID, senderType, senderID, content string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}

	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		return err
	}

	_, err = s.repo.InsertMessage(ctx, db.InsertMessageParams{
		SessionID:  sid,
		SenderType: senderType,
		SenderID:   senderUUID,
		Content:    content,
	})
	return err
}

func (s *Service) UpdateSessionStatus(ctx context.Context, sessionID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}

	return s.repo.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		ID:     sid,
		Status: sql.NullString{String: "ENDED", Valid: true},
	})
}
