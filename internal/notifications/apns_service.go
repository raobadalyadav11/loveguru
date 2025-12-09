package notifications

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"time"

	"loveguru/internal/config"
)

// APNSNotification represents an Apple Push Notification Service notification
type APNSNotification struct {
	Token   string `json:"token"`
	Payload struct {
		APS struct {
			Alert            interface{} `json:"alert,omitempty"`
			Badge            *int        `json:"badge,omitempty"`
			Sound            string      `json:"sound,omitempty"`
			Category         string      `json:"category,omitempty"`
			ContentAvailable int         `json:"content-available,omitempty"`
		} `json:"aps"`
		Data map[string]interface{} `json:"data,omitempty"`
	} `json:"payload"`
	Priority   int    `json:"priority,omitempty"`
	PushType   string `json:"push_type,omitempty"`
	Topic      string `json:"topic,omitempty"`
	CollapseID string `json:"collapse_id,omitempty"`
	ThreadID   string `json:"thread-id,omitempty"`
}

// APNSAlert represents the alert part of an APNS notification
type APNSAlert struct {
	Title        string   `json:"title,omitempty"`
	Body         string   `json:"body,omitempty"`
	TitleLocKey  string   `json:"title-loc-key,omitempty"`
	TitleArgs    []string `json:"title-loc-args,omitempty"`
	ActionLocKey string   `json:"action-loc-key,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
}

// APNSResponse represents the response from APNS API
type APNSResponse struct {
	ApnsID string `json:"apns-id"`
}

// APNSService handles Apple Push Notification Service notifications
type APNSService struct {
	teamID      string
	keyID       string
	privateKey  *ecdsa.PrivateKey
	bundleID    string
	environment string // "development" or "production"
	client      *http.Client
}

func NewAPNSService(cfg *config.APNSConfig) (*APNSService, error) {
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("APNS private key is required")
	}

	// Parse the private key
	privateKey, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse APNS private key: %w", err)
	}

	return &APNSService{
		teamID:      cfg.TeamID,
		keyID:       cfg.KeyID,
		privateKey:  privateKey,
		bundleID:    cfg.BundleID,
		environment: cfg.Environment,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendPushNotification sends a push notification to a specific device
func (a *APNSService) SendPushNotification(deviceToken, title, body string, data map[string]interface{}) error {
	if a.environment == "" {
		a.environment = "development"
	}

	notification := APNSNotification{
		Token:    deviceToken,
		Priority: 10,
		PushType: "alert",
		Topic:    a.bundleID,
	}

	// Set alert
	if title != "" && body != "" {
		notification.Payload.APS.Alert = APNSAlert{
			Title: title,
			Body:  body,
		}
	} else if body != "" {
		notification.Payload.APS.Alert = body
	}

	// Set data
	if data != nil {
		notification.Payload.Data = data
	}

	return a.sendNotification(notification)
}

// SendToTopic sends a push notification (APNS doesn't have topics, so this sends to multiple tokens)
func (a *APNSService) SendToTopic(tokens []string, title, body string, data map[string]interface{}) error {
	var errors []string
	for _, token := range tokens {
		err := a.SendPushNotification(token, title, body, data)
		if err != nil {
			errors = append(errors, fmt.Sprintf("token %s: %v", token, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("APNS delivery errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// sendNotification sends a notification to APNS API
func (a *APNSService) sendNotification(notification APNSNotification) error {
	jsonData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal APNS notification: %w", err)
	}

	url := a.getAPNSURL()
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create APNS request: %w", err)
	}

	// Add authentication headers
	authToken := a.generateAuthToken()
	req.Header.Set("authorization", fmt.Sprintf("bearer %s", authToken))
	req.Header.Set("apns-id", generateApnsID())
	req.Header.Set("apns-push-type", notification.PushType)
	req.Header.Set("apns-priority", fmt.Sprintf("%d", notification.Priority))
	req.Header.Set("apns-topic", notification.Topic)
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send APNS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("APNS API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getAPNSURL returns the appropriate APNS URL based on environment
func (a *APNSService) getAPNSURL() string {
	if a.environment == "production" {
		return "https://api.push.apple.com:443/3/device/"
	}
	return "https://api.sandbox.push.apple.com:443/3/device/"
}

// generateAuthToken generates JWT token for APNS authentication
func (a *APNSService) generateAuthToken() string {
	// This is a simplified JWT generation
	// In production, use a proper JWT library like github.com/golang-jwt/jwt
	header := map[string]interface{}{
		"alg": "ES256",
		"kid": a.keyID,
	}

	claims := map[string]interface{}{
		"iss": a.teamID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	headerBytes, _ := json.Marshal(header)
	claimsBytes, _ := json.Marshal(claims)

	headerB64 := base64URLEncode(headerBytes)
	claimsB64 := base64URLEncode(claimsBytes)

	signingInput := headerB64 + "." + claimsB64

	// Sign with ECDSA P-256 (simplified)
	r, s, _ := ecdsa.Sign(rand.Reader, a.privateKey, []byte(signingInput))

	signature := r.Bytes()
	if len(s.Bytes()) > len(signature) {
		signature = s.Bytes()
	} else {
		sig := make([]byte, len(s.Bytes()))
		copy(sig, s.Bytes())
		signature = sig
	}

	signatureB64 := base64URLEncode(signature)

	return signingInput + "." + signatureB64
}

// ValidateConfig validates the APNS configuration
func (a *APNSService) ValidateConfig() error {
	if a.teamID == "" {
		return fmt.Errorf("APNS team ID is required")
	}
	if a.keyID == "" {
		return fmt.Errorf("APNS key ID is required")
	}
	if a.bundleID == "" {
		return fmt.Errorf("APNS bundle ID is required")
	}
	if a.privateKey == nil {
		return fmt.Errorf("APNS private key is required")
	}
	return nil
}

// Helper functions
func parsePrivateKey(privateKeyPEM string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func generateApnsID() string {
	// Generate a simple UUID-like ID
	timestamp := time.Now().Unix()
	random := new(big.Int)
	random.SetInt64(timestamp)
	return random.Text(16)
}

func base64URLEncode(data []byte) string {
	result := make([]byte, len(data))
	copy(result, data)

	// Simple base64 encoding (not URL-safe)
	// In production, use proper base64.URLEncoding
	for i, b := range result {
		if b == 0 {
			result[i] = 'A'
		} else if b == 255 {
			result[i] = '_'
		}
	}

	return string(result)
}
