package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"loveguru/internal/config"
)

// FCMMessage represents a Firebase Cloud Messaging notification
type FCMMessage struct {
	To           string                 `json:"to"`
	Topic        string                 `json:"topic,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Notification struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Image string `json:"image,omitempty"`
	} `json:"notification"`
	Priority   string `json:"priority,omitempty"`
	TimeToLive int    `json:"time_to_live,omitempty"`
}

// FCMResponse represents the response from FCM API
type FCMResponse struct {
	SuccessCount int   `json:"success_count"`
	FailureCount int   `json:"failure_count"`
	CanonicalIDs int   `json:"canonical_ids"`
	MulticastID  int64 `json:"multicast_id"`
	Results      []struct {
		MessageID      string `json:"message_id"`
		RegistrationID string `json:"registration_id"`
		Error          string `json:"error"`
	} `json:"results"`
}

// FCMService handles Firebase Cloud Messaging notifications
type FCMService struct {
	serverKey string
	projectID string
	client    *http.Client
}

func NewFCMService(cfg *config.FCMConfig) *FCMService {
	return &FCMService{
		serverKey: cfg.ServerKey,
		projectID: cfg.ProjectID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendPushNotification sends a push notification to a specific device
func (f *FCMService) SendPushNotification(deviceToken, title, body string, data map[string]interface{}) error {
	if f.serverKey == "" {
		return fmt.Errorf("FCM server key not configured")
	}

	message := FCMMessage{
		To: deviceToken,
		Notification: struct {
			Title string `json:"title"`
			Body  string `json:"body"`
			Image string `json:"image,omitempty"`
		}{
			Title: title,
			Body:  body,
		},
		Data:     data,
		Priority: "high",
	}

	return f.sendMessage(message)
}

// SendToTopic sends a push notification to all devices subscribed to a topic
func (f *FCMService) SendToTopic(topic, title, body string, data map[string]interface{}) error {
	if f.serverKey == "" {
		return fmt.Errorf("FCM server key not configured")
	}

	message := FCMMessage{
		Topic: topic,
		Notification: struct {
			Title string `json:"title"`
			Body  string `json:"body"`
			Image string `json:"image,omitempty"`
		}{
			Title: title,
			Body:  body,
		},
		Data:     data,
		Priority: "normal",
	}

	return f.sendMessage(message)
}

// sendMessage sends a message to FCM API
func (f *FCMService) sendMessage(message FCMMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM message: %w", err)
	}

	url := "https://fcm.googleapis.com/fcm/send"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create FCM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+f.serverKey)

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send FCM request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FCM API returned status %d", resp.StatusCode)
	}

	var fcmResp FCMResponse
	if err := json.NewDecoder(resp.Body).Decode(&fcmResp); err != nil {
		return fmt.Errorf("failed to decode FCM response: %w", err)
	}

	if fcmResp.FailureCount > 0 {
		return fmt.Errorf("FCM delivery failed for %d messages", fcmResp.FailureCount)
	}

	return nil
}

// ValidateConfig validates the FCM configuration
func (f *FCMService) ValidateConfig() error {
	if f.serverKey == "" {
		return fmt.Errorf("FCM server key is required")
	}
	if f.projectID == "" {
		return fmt.Errorf("FCM project ID is required")
	}
	return nil
}
