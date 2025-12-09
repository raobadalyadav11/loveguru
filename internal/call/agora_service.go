package call

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"loveguru/internal/config"

	"github.com/google/uuid"
)

type AgoraService struct {
	config *config.AgoraConfig
}

type AgoraCallInfo struct {
	Token      string
	RoomID     string
	AppID      string
	Channel    string
	UID        string
	ExternalID string
}

type AgoraCallOptions struct {
	Role            uint32
	PrivilegeExpire uint32
	ChannelName     string
}

func NewAgoraService(agoraConfig *config.AgoraConfig) *AgoraService {
	return &AgoraService{
		config: agoraConfig,
	}
}

func (s *AgoraService) CreateCallSession(ctx context.Context, userID, advisorID string) (*AgoraCallInfo, error) {
	if s.config.AppID == "" || s.config.AppCert == "" {
		return nil, fmt.Errorf("Agora credentials not configured")
	}

	// Generate unique room/channel name
	channelName := fmt.Sprintf("call_%s_%s", userID, advisorID)

	// Generate unique external ID for this call
	externalID := uuid.New().String()

	// Generate UID for the user (using user ID hash)
	uid := s.generateUID(userID)

	// Create token with 1 hour expiry
	expireTime := uint32(time.Now().Add(time.Duration(s.config.TokenTTL) * time.Second).Unix())

	// Generate RTC token for voice call
	token, err := s.generateAgoraToken(channelName, uid, expireTime)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Agora token: %w", err)
	}

	return &AgoraCallInfo{
		Token:      token,
		RoomID:     externalID,
		AppID:      s.config.AppID,
		Channel:    channelName,
		UID:        fmt.Sprintf("%d", uid),
		ExternalID: externalID,
	}, nil
}

func (s *AgoraService) GenerateUserToken(ctx context.Context, channelName, userID string) (string, error) {
	if s.config.AppID == "" || s.config.AppCert == "" {
		return "", fmt.Errorf("Agora credentials not configured")
	}

	uid := s.generateUID(userID)
	expireTime := uint32(time.Now().Add(time.Duration(s.config.TokenTTL) * time.Second).Unix())

	token, err := s.generateAgoraToken(channelName, uid, expireTime)
	if err != nil {
		return "", fmt.Errorf("failed to generate user token: %w", err)
	}

	return token, nil
}

func (s *AgoraService) EndCall(ctx context.Context, externalCallID string) error {
	// Agora doesn't require explicit end call API calls
	// The call will end when all users leave the channel
	// We can perform cleanup operations here if needed
	return nil
}

func (s *AgoraService) GetCallStats(ctx context.Context, externalCallID string) (duration int, status string, err error) {
	// Agora doesn't provide direct call stats via API
	// In a production environment, you might want to:
	// 1. Track call start/end times in your database
	// 2. Use Agora Webhooks for call events
	// 3. Query Agora Cloud Recording service for detailed stats

	// For now, return estimated values that would be tracked in your database
	return 0, "ENDED", nil
}

// generateUID creates a consistent UID from a user ID string
func (s *AgoraService) generateUID(userID string) uint32 {
	// Simple hash function to convert user ID to UID
	// In production, you might want a more sophisticated approach
	hash := 0
	for _, char := range userID {
		hash = hash*31 + int(char)
	}

	// Ensure UID is positive and within Agora's limits (0 to 2^32-1)
	if hash < 0 {
		hash = -hash
	}

	return uint32(hash % 1000000000) // Keep it under 1 billion
}

// generateAgoraToken creates a basic Agora token (simplified version)
// In production, you would use the official Agora SDK
func (s *AgoraService) generateAgoraToken(channelName string, uid uint32, expireTime uint32) (string, error) {
	// This is a simplified token generation
	// In production, use proper HMAC-SHA256 with Agora's specific algorithm
	// This demonstrates the integration pattern without external dependencies

	// Create a basic token payload
	payload := map[string]interface{}{
		"app_id":      s.config.AppID,
		"channel":     channelName,
		"uid":         uid,
		"privilege":   1, // Publisher privilege
		"expire_time": expireTime,
		"timestamp":   time.Now().Unix(),
	}

	// Serialize payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Create HMAC signature (simplified - not the real Agora algorithm)
	mac := hmac.New(sha256.New, []byte(s.config.AppCert))
	mac.Write(payloadJSON)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Combine payload and signature
	token := base64.StdEncoding.EncodeToString([]byte(string(payloadJSON) + "." + signature))

	return token, nil
}

// ValidateConfig validates the Agora configuration
func (s *AgoraService) ValidateConfig() error {
	if s.config.AppID == "" {
		return fmt.Errorf("Agora App ID is required")
	}
	if s.config.AppCert == "" {
		return fmt.Errorf("Agora App Certificate is required")
	}
	return nil
}
