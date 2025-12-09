package chat

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/internal/notifications"
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

func (s *Service) InsertMessageWithID(ctx context.Context, sessionID, senderType, senderID, content string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *Service) UpdateMessageReadStatus(ctx context.Context, messageID, readerID string) error {
	return errors.New("not implemented")
}

func (s *Service) GetSessionParticipants(ctx context.Context, sessionID string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) UpdateSessionStatusWithNotification(ctx context.Context, sessionID, status, userID string) error {
	return errors.New("not implemented")
}

func (s *Service) GetActiveSessions(ctx context.Context, userID string) ([]db.Session, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) GetSessionAnalytics(ctx context.Context, userID string) (*SessionAnalytics, error) {
	return nil, errors.New("not implemented")
}

// sendPushNotificationForMessage sends push notifications to other session participants
func (s *Service) sendPushNotificationForMessage(ctx context.Context, sessionID, senderID, content string) {
	// Get device tokens for other participants
	deviceTokens, err := s.getDeviceTokensForSession(sessionID, senderID)
	if err != nil {
		log.Printf("Error getting device tokens: %v", err)
		return
	}

	if len(deviceTokens) == 0 {
		return // No devices to notify
	}

	// Get sender name for notification
	senderName, err := s.getUserDisplayName(ctx, senderID)
	if err != nil {
		log.Printf("Error getting sender name: %v", err)
		senderName = "Someone"
	}

	// Prepare message content (truncate if too long)
	notificationContent := content
	if len(notificationContent) > 50 {
		notificationContent = notificationContent[:50] + "..."
	}

	// Send push notification
	notificationService := notifications.NewNotificationService()
	err = notificationService.SendChatNotification(deviceTokens, senderName, notificationContent, sessionID)
	if err != nil {
		log.Printf("Error sending push notification: %v", err)
	}
}

// getDeviceTokensForSession gets device tokens for all participants except the sender
func (s *Service) getDeviceTokensForSession(sessionID, excludeUserID string) ([]string, error) {
	// This would need to be implemented with proper queries
	// For now, return empty slice - to be implemented with the database queries
	return []string{}, nil
}

// getUserDisplayName gets the display name for a user
func (s *Service) getUserDisplayName(ctx context.Context, userID string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}

	user, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		return "", err
	}

	return user.DisplayName, nil
}

// SendMessageWithNotification sends a message and triggers push notifications
func (s *Service) SendMessageWithNotification(ctx context.Context, sessionID, senderType, senderID, content string) (string, error) {
	// Insert message and get ID
	messageID, err := s.InsertMessageWithID(ctx, sessionID, senderType, senderID, content)
	if err != nil {
		return "", err
	}

	// Send push notification asynchronously
	go s.sendPushNotificationForMessage(ctx, sessionID, senderID, content)

	return messageID, nil
}

type SessionAnalytics struct {
	TotalSessions     int32
	CompletedSessions int32
	AverageDuration   float64
	CompletionRate    float64
}
