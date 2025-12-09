# Push Notifications Setup Guide

This guide explains how to configure and set up FCM/APNS push notifications in the LoveGuru application for real-time alerts on chat messages and call requests.

## Overview

The notification service has been enhanced with real push notification capabilities. The system now:

- ✅ **FCM Integration**: Firebase Cloud Messaging for Android and web
- ✅ **APNS Integration**: Apple Push Notification Service for iOS
- ✅ **Real-time Alerts**: Instant notifications for chat messages and call requests
- ✅ **Session Updates**: Notifications for session status changes
- ✅ **Multi-platform**: Supports both Android and iOS devices
- ✅ **Graceful Fallbacks**: Works even if some services are not configured
- ✅ **Configuration Validation**: Startup checks and status reporting

## Prerequisites

### Firebase Cloud Messaging (FCM)
1. **Firebase Account**: Create at [Firebase Console](https://console.firebase.google.com)
2. **Firebase Project**: Create a new project for your app
3. **Server Key**: Get from Project Settings > Cloud Messaging
4. **Project ID**: Your Firebase project identifier

### Apple Push Notification Service (APNS)
1. **Apple Developer Account**: Required for APNS access
2. **App ID**: Create an App ID in Apple Developer Console
3. **Push Certificates**: Generate APNs certificates or keys
4. **Team ID**: Your Apple Developer Team identifier
5. **Key ID**: ID of your APNs authentication key

## Configuration

### 1. Environment Variables

Add the following to your `.env` file:

```bash
# Firebase Cloud Messaging (FCM)
FCM_SERVER_KEY=your_fcm_server_key_here
FCM_PROJECT_ID=your_fcm_project_id_here

# Apple Push Notification Service (APNS)
APNS_TEAM_ID=your_apns_team_id_here
APNS_KEY_ID=your_apns_key_id_here
APNS_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
Your APNS private key here
-----END PRIVATE KEY-----"
APNS_BUNDLE_ID=com.yourcompany.loveguru
APNS_ENVIRONMENT=development

# Email Configuration (enhanced)
EMAIL_FROM=your_email@example.com
EMAIL_PASS=your_email_password_or_app_password
EMAIL_HOST=smtp.gmail.com
EMAIL_PORT=587
```

### 2. Config File

Add to your `config.yaml`:

```yaml
fcm:
  server_key: "your_fcm_server_key_here"
  project_id: "your_fcm_project_id_here"

apns:
  team_id: "your_apns_team_id_here"
  key_id: "your_apns_key_id_here"
  private_key: |
    -----BEGIN PRIVATE KEY-----
    Your APNS private key here
    -----END PRIVATE KEY-----
  bundle_id: "com.yourcompany.loveguru"
  environment: "development"

email:
  from: "your_email@example.com"
  password: "your_email_password_or_app_password"
  host: "smtp.gmail.com"
  port: "587"
```

## Testing the Integration

### 1. Validate Configuration

Run the server and check for notification status messages:
```
FCM push notifications enabled
APNS push notifications enabled
```

Or warnings:
```
Warning: No push notification services configured. Push notifications will not work.
Warning: FCM service enabled but not configured properly
```

### 2. Test Push Notification

```go
// Example usage in your code
notificationService := notifications.NewNotificationServiceWithConfig(cfg)

// Send chat notification
deviceTokens := []string{"device_token_here"}
err := notificationService.SendChatNotification(
    deviceTokens,
    "John Doe",
    "Hey, how are you?",
    "session-123",
)
if err != nil {
    log.Printf("Failed to send notification: %v", err)
}
```

### 3. Test Status Check

```go
status := notificationService.GetPushNotificationStatus()
fmt.Printf("FCM enabled: %t\n", status["fcm_enabled"])
fmt.Printf("APNS enabled: %t\n", status["apns_enabled"])
```

## Integration Details

### Service Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │    │ Notification    │    │ Push Services   │
│                 │    │ Service         │    │                 │
│ Chat Messages   │───▶│ SendChatNotify  │───▶│ FCM API         │
│ Call Requests   │───▶│ SendCallNotify  │───▶│ APNS API        │
│ Session Updates │───▶│ SessionUpdates  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   Database      │
                       │                 │
                       │ User Tokens     │
                       │ Device Registry │
                       └─────────────────┘
```

### API Usage

#### Basic Push Notification

```go
func SendPushToDevice(deviceTokens []string, title, body string, data map[string]interface{}) error {
    return notificationService.SendPushNotification(
        deviceTokens,
        "all", // platform
        title,
        body,
        data,
    )
}
```

#### Chat Notifications

```go
func NotifyNewMessage(sessionID, senderName, message string, deviceTokens []string) error {
    return notificationService.SendChatNotification(
        deviceTokens,
        senderName,
        message,
        sessionID,
    )
}
```

#### Call Notifications

```go
func NotifyIncomingCall(callerName, callType, sessionID string, deviceTokens []string) error {
    return notificationService.SendCallNotification(
        deviceTokens,
        callerName,
        callType, // "voice" or "video"
        sessionID,
    )
}
```

#### Session Status Updates

```go
func NotifySessionUpdate(sessionID, advisorName, action string, deviceTokens []string) error {
    return notificationService.SendSessionUpdateNotification(
        deviceTokens,
        advisorName,
        sessionID,
        action, // "started", "ended", "accepted", "rejected"
    )
}
```

## Device Token Management

### Token Storage

Store device tokens in your database:

```sql
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    device_token TEXT NOT NULL,
    platform TEXT NOT NULL, -- 'android' or 'ios'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Token Validation

```go
func ValidateAndStoreToken(userID, deviceToken, platform string) error {
    if !notificationService.ValidateDeviceToken(deviceToken) {
        return fmt.Errorf("invalid device token")
    }
    
    // Store in database
    // Update user's active devices
    return nil
}
```

## Frontend Integration

### Android (FCM)

```javascript
// Get FCM token
import messaging from '@react-native-firebase/messaging';

const getFCMToken = async () => {
  const token = await messaging().getToken();
  console.log('FCM Token:', token);
  
  // Send token to your backend
  await sendTokenToServer(token, 'android');
};
```

### iOS (APNS)

```javascript
import { Notifications } from 'react-native-notifications';

const getAPNSToken = async () => {
  Notifications.events().registerRemoteNotificationsRegistered((event) => {
    const token = event.deviceToken;
    console.log('APNS Token:', token);
    
    // Send token to your backend
    sendTokenToServer(token, 'ios');
  });
  
  Notifications.registerRemoteNotifications();
};
```

### Web (FCM)

```javascript
importScripts('https://www.gstatic.com/firebasejs/9.0.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/9.0.0/firebase-messaging-compat.js');

firebase.initializeApp({
  apiKey: "your-api-key",
  projectId: "your-project-id",
});

const messaging = firebase.messaging();

messaging.onMessage((payload) => {
  console.log('Message received:', payload);
  // Handle notification
});
```

## Notification Types

### Chat Notifications

```json
{
  "title": "New Message",
  "body": "John Doe: Hey, how are you?",
  "data": {
    "type": "chat",
    "session_id": "session-123",
    "sender": "John Doe",
    "message": "Hey, how are you?"
  }
}
```

### Call Notifications

```json
{
  "title": "Incoming Call",
  "body": "John Doe is calling you for a voice session",
  "data": {
    "type": "call",
    "session_id": "session-123",
    "caller": "John Doe",
    "call_type": "voice"
  }
}
```

### Session Updates

```json
{
  "title": "Session Started",
  "body": "Your session with Jane Smith has begun",
  "data": {
    "type": "session",
    "session_id": "session-123",
    "advisor": "Jane Smith",
    "action": "started"
  }
}
```

## Error Handling

### Common Issues

1. **"FCM server key not configured"**
   - Check FCM_SERVER_KEY environment variable
   - Verify Firebase project setup

2. **"APNS private key invalid"**
   - Check APNS_PRIVATE_KEY format (must include PEM headers)
   - Verify key matches your Apple Developer account

3. **"Device token invalid"**
   - Token format should be 64 characters (APNS) or 152+ (FCM)
   - Check token hasn't expired or been revoked

### Fallback Strategy

```go
func sendNotificationWithFallback(deviceTokens []string, title, body string) error {
    // Try push notification
    err := notificationService.SendPushNotification(deviceTokens, "all", title, body, nil)
    if err != nil {
        // Fallback to email or SMS
        log.Printf("Push notification failed: %v", err)
        // Send email notification instead
        return sendEmailNotification(deviceTokens, title, body)
    }
    return nil
}
```

## Production Deployment

### Security Checklist

- [ ] Use environment variables for all credentials
- [ ] Implement token rotation for APNS keys
- [ ] Set up proper error monitoring and alerting
- [ ] Implement rate limiting to prevent abuse
- [ ] Use production APNS environment for App Store builds

### Monitoring and Analytics

```go
// Track notification delivery rates
type NotificationMetrics struct {
    TotalSent    int
    SuccessCount int
    FCMFailures  int
    APNSFailures int
}

func trackNotificationMetrics(metrics NotificationMetrics) {
    // Send to monitoring service
    // Prometheus metrics
    // Custom analytics
}
```

### Rate Limiting

```go
func (n *NotificationService) SendNotificationWithRateLimit(deviceTokens []string, title, body string) error {
    // Check rate limits per user/device
    if n.isRateLimited(deviceTokens[0]) {
        return fmt.Errorf("rate limited")
    }
    
    // Send notification
    err := n.SendPushNotification(deviceTokens, "all", title, body, nil)
    
    // Update rate limiting
    n.updateRateLimit(deviceTokens[0])
    
    return err
}
```

## Troubleshooting

### Debug Commands

```bash
# Test FCM connectivity
curl -X POST https://fcm.googleapis.com/fcm/send \
  -H "Authorization: key=YOUR_SERVER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":"DEVICE_TOKEN","notification":{"title":"Test","body":"Test message"}}'

# Check server logs for notification errors
tail -f logs/application.log | grep -i notification
```

### Common Error Messages

1. **"InvalidDeviceToken"**: Device token has expired or is invalid
2. **"Unregistered"**: App has been uninstalled from device
3. **"DeviceTokenNotForTopic"**: Token doesn't match topic subscription
4. **"APNS authentication error"**: APNS credentials are invalid

## Advanced Features

### Notification Groups

```go
// Group similar notifications to prevent spam
func (n *NotificationService) SendGroupedNotification(userID, groupKey, title, body string) error {
    // Check if user already has recent notification in this group
    if n.hasRecentNotification(userID, groupKey, 5*time.Minute) {
        return nil // Skip to prevent spam
    }
    
    // Send notification and record in cache
    deviceTokens := n.getUserDeviceTokens(userID)
    return n.SendPushNotification(deviceTokens, "all", title, body, map[string]interface{}{
        "group": groupKey,
    })
}
```

### Rich Notifications

```go
// iOS rich notifications with actions
func (n *NotificationService) SendRichNotification(deviceTokens []string, title, body string) error {
    // For APNS, you can include action buttons and categories
    // This requires additional configuration in your iOS app
    return n.apns.SendPushNotification(deviceTokens, title, body, map[string]interface{}{
        "category": "MESSAGE_CATEGORY",
        "actions": []string{"Reply", "View"},
    })
}
```

### Delivery Tracking

```go
type NotificationResult struct {
    Token      string
    Success    bool
    Error      string
    Timestamp  time.Time
}

func (n *NotificationService) SendWithTracking(deviceTokens []string, title, body string) []NotificationResult {
    var results []NotificationResult
    
    for _, token := range deviceTokens {
        result := NotificationResult{
            Token:     token,
            Timestamp: time.Now(),
        }
        
        err := n.SendPushNotification([]string{token}, "all", title, body, nil)
        if err != nil {
            result.Success = false
            result.Error = err.Error()
        } else {
            result.Success = true
        }
        
        results = append(results, result)
    }
    
    // Store results for analytics
    n.storeDeliveryResults(results)
    
    return results
}
```

## Configuration Validation

The notification service performs startup validation:

```go
func validateNotificationConfig(cfg *config.Config) error {
    var errors []string
    
    // Validate FCM
    if cfg.FCM.ServerKey != "" && cfg.FCM.ProjectID == "" {
        errors = append(errors, "FCM project ID required when server key is provided")
    }
    
    // Validate APNS
    if cfg.APNS.PrivateKey != "" {
        if cfg.APNS.TeamID == "" || cfg.APNS.KeyID == "" || cfg.APNS.BundleID == "" {
            errors = append(errors, "Complete APNS configuration required")
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("notification config errors: %s", strings.Join(errors, "; "))
    }
    
    return nil
}
```

## Next Steps

1. ✅ **Complete**: Implement FCM/APNS push notification services
2. 🔄 **Optional**: Add notification delivery analytics dashboard
3. 🔄 **Optional**: Implement notification scheduling and batching
4. 🔄 **Optional**: Add rich notification support with images and actions
5. 🔄 **Optional**: Implement notification preferences and user controls

---

**Integration Status**: ✅ Complete

The notification service now provides real push notification capabilities using FCM and APNS. The system is ready for production use with comprehensive error handling and multi-platform support.