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

	err = s.repo.EndCall(ctx, sid)
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

// UpdateCallStatus updates the call status (RINGING, CONNECTED, ENDED)
func (s *Service) UpdateCallStatus(ctx context.Context, sessionID, status string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}

	// Validate status
	validStatuses := []string{"RINGING", "CONNECTED", "ENDED"}
	isValid := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid call status: %s", status)
	}

	// Update call status in database using generated query
	err = s.repo.UpdateCallStatus(ctx, db.UpdateCallStatusParams{
		StatusUpdate: sql.NullString{String: status, Valid: true},
		SessionID:    sid,
	})
	if err != nil {
		return err
	}

	return nil
}

// EndCall marks the call as ended and creates final call log
func (s *Service) EndCallWithStatus(ctx context.Context, sessionID string) error {
	err := s.UpdateCallStatus(ctx, sessionID, "ENDED")
	if err != nil {
		return err
	}

	return nil
}

// CreateFeedbackPrompt creates a feedback prompt for a completed call
func (s *Service) CreateFeedbackPrompt(ctx context.Context, sessionID, userID, advisorID string) (string, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return "", err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}

	aid, err := uuid.Parse(advisorID)
	if err != nil {
		return "", err
	}

	// Create feedback prompt
	promptID, err := s.repo.CreateFeedbackPrompt(ctx, db.CreateFeedbackPromptParams{
		SessionID: sid,
		UserID:    uid,
		AdvisorID: aid,
	})
	if err != nil {
		return "", err
	}

	return promptID.String(), nil
}

// GetPendingFeedbackPrompts gets all pending feedback prompts
func (s *Service) GetPendingFeedbackPrompts(ctx context.Context) ([]FeedbackPrompt, error) {
	prompts, err := s.repo.GetPendingFeedbackPrompts(ctx)
	if err != nil {
		return nil, err
	}

	var feedbackPrompts []FeedbackPrompt
	for _, p := range prompts {
		feedbackPrompts = append(feedbackPrompts, FeedbackPrompt{
			ID:           p.ID.String(),
			SessionID:    p.SessionID.String(),
			UserName:     p.UserName,
			AdvisorName:  p.AdvisorName,
			PromptSentAt: p.PromptSentAt.Time.Format("2006-01-02T15:04:05Z"),
		})
	}

	return feedbackPrompts, nil
}

// SubmitFeedback submits user feedback for a call
func (s *Service) SubmitFeedback(ctx context.Context, promptID string, rating int, feedbackText string) error {
	pid, err := uuid.Parse(promptID)
	if err != nil {
		return err
	}

	// Validate rating
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	err = s.repo.SubmitFeedback(ctx, db.SubmitFeedbackParams{
		ID:           pid,
		Rating:       sql.NullInt32{Int32: int32(rating), Valid: true},
		FeedbackText: sql.NullString{String: feedbackText, Valid: true},
	})
	if err != nil {
		return err
	}

	return nil
}

// AutoPromptFeedback automatically creates feedback prompts for recently ended calls
func (s *Service) AutoPromptFeedback(ctx context.Context) error {
	// This would typically be called by a background job
	// For now, we'll implement the logic to find calls that ended recently but don't have feedback prompts

	// Get recent call sessions that ended
	recentSessions, err := s.repo.GetRecentEndedSessions(ctx)
	if err != nil {
		return err
	}

	for _, session := range recentSessions {
		// Check if feedback prompt already exists
		existingPrompt, err := s.repo.GetFeedbackPromptBySession(ctx, session.ID)
		if err != nil {
			continue // Skip if error getting existing prompt
		}

		if existingPrompt.ID != uuid.Nil {
			continue // Feedback prompt already exists
		}

		// Create feedback prompt
		_, err = s.CreateFeedbackPrompt(ctx, session.ID.String(), session.UserID.String(), session.AdvisorID.UUID.String())
		if err != nil {
			// Log error but continue with other sessions
			continue
		}
	}

	return nil
}

type FeedbackPrompt struct {
	ID           string
	SessionID    string
	UserName     string
	AdvisorName  string
	PromptSentAt string
}

// GetCallStatus gets the current call status
func (s *Service) GetCallStatus(ctx context.Context, sessionID string) (string, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return "", err
	}

	// Get call status from database using generated method
	callStatus, err := s.repo.GetCallStatus(ctx, sid)
	if err != nil {
		return "", err
	}

	return callStatus.StatusUpdate.String, nil
}

// StartCall initiates a call and sets status to RINGING
func (s *Service) StartCall(ctx context.Context, sessionID string) error {
	err := s.UpdateCallStatus(ctx, sessionID, "RINGING")
	if err != nil {
		return err
	}

	// TODO: Notify the callee about incoming call
	// This would integrate with the notification system

	return nil
}

// ConnectCall marks the call as connected
func (s *Service) ConnectCall(ctx context.Context, sessionID string) error {
	err := s.UpdateCallStatus(ctx, sessionID, "CONNECTED")
	if err != nil {
		return err
	}

	return nil
}
