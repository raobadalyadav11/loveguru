# LoveGuru Backend API Documentation

## Overview

This document provides comprehensive API documentation for the LoveGuru backend service. The API is built using gRPC with HTTP/JSON fallback for WebSocket connections.

## Base URL

- **gRPC Server**: `localhost:50051`
- **HTTP/WebSocket Server**: `localhost:8080`
- **Environment**: Development

## Authentication

All protected endpoints require JWT token authentication. Include the token in the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

## Services

### 1. Authentication Service

#### Register User
```protobuf
message RegisterRequest {
  string email = 1;
  string phone = 2;
  string password = 3;
  string display_name = 4;
  Role role = 5; // USER, ADVISOR, ADMIN
}

message RegisterResponse {
  User user = 1;
  Tokens tokens = 2;
}
```

**Example Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123",
  "display_name": "John Doe",
  "role": "USER"
}
```

#### Login
```protobuf
message LoginRequest {
  string email = 1;
  string phone = 2;
  string password = 3;
}

message LoginResponse {
  Tokens tokens = 1;
}
```

#### Refresh Token
```protobuf
message RefreshRequest {
  string refresh_token = 1;
}

message RefreshResponse {
  Tokens tokens = 1;
}
```

#### Logout
```protobuf
message LogoutRequest {}
message LogoutResponse {
  bool success = 1;
}
```

### 2. User Service

#### Get Profile
```protobuf
message GetProfileRequest {}

message GetProfileResponse {
  User user = 1;
}
```

#### Update Profile
```protobuf
message UpdateProfileRequest {
  string display_name = 1;
  Gender gender = 2;
  string dob = 3;
}

message UpdateProfileResponse {
  User user = 1;
}
```

#### Get User Sessions
```protobuf
message GetSessionsRequest {
  int32 limit = 1;
  int32 offset = 2;
}

message GetSessionsResponse {
  repeated Session sessions = 1;
}
```

### 3. Advisor Service

#### List Advisors
```protobuf
message ListAdvisorsRequest {
  double rating_min = 1;
  int32 experience_min = 2;
  repeated string languages = 3;
  repeated string specializations = 4;
  AdvisorStatus status = 5;
  string search = 6;
  string sort = 7; // top_rated, price, experience
  int32 limit = 8;
  int32 offset = 9;
}

message ListAdvisorsResponse {
  repeated AdvisorWithRating advisors = 1;
}
```

#### Get Advisor Details
```protobuf
message GetAdvisorRequest {
  string id = 1;
}

message GetAdvisorResponse {
  AdvisorWithRating advisor = 1;
}
```

#### Apply as Advisor
```protobuf
message ApplyAsAdvisorRequest {
  string bio = 1;
  int32 experience_years = 2;
  repeated string languages = 3;
  repeated string specializations = 4;
  double hourly_rate = 5;
}

message ApplyAsAdvisorResponse {
  common.Advisor advisor = 1;
}
```

#### Update Advisor Profile
```protobuf
message UpdateProfileRequest {
  string bio = 1;
  int32 experience_years = 2;
  repeated string languages = 3;
  repeated string specializations = 4;
  double hourly_rate = 5;
  AdvisorStatus status = 6;
}

message UpdateProfileResponse {
  common.Advisor advisor = 1;
}
```

### 4. Chat Service

#### Create Chat Session
```protobuf
message CreateSessionRequest {
  string advisor_id = 1;
  SessionType type = 2; // CHAT, CALL, AI_CHAT
}

message CreateSessionResponse {
  Session session = 1;
}
```

#### Get Messages
```protobuf
message GetMessagesRequest {
  string session_id = 1;
  int32 limit = 2;
  int32 offset = 3;
}

message GetMessagesResponse {
  repeated ChatMessage messages = 1;
}
```

#### WebSocket Chat
**Endpoint**: `ws://localhost:8080/ws/chat`

**Query Parameters**:
- `session_id`: The chat session ID
- `token`: JWT authentication token

**Message Format**:
```json
{
  "type": "MESSAGE",
  "session_id": "uuid",
  "content": "Hello, I need advice!"
}
```

**Response Format**:
```json
{
  "type": "MESSAGE",
  "session_id": "uuid",
  "sender_id": "user-uuid",
  "content": "Hello! How can I help you?",
  "timestamp": "2023-12-01T10:00:00Z"
}
```

### 5. Call Service

#### Create Call Session
```protobuf
message CreateSessionRequest {
  string advisor_id = 1;
}

message CreateSessionResponse {
  Session session = 1;
  string call_token = 2;
  string room_id = 3;
}
```

#### End Call
```protobuf
message EndCallRequest {
  string session_id = 1;
}

message EndCallResponse {
  bool success = 1;
}
```

#### Get Call Details
```protobuf
message GetCallRequest {
  string session_id = 1;
}

message GetCallResponse {
  Session session = 1;
}
```

### 6. AI Service

#### Chat with AI
```protobuf
message ChatRequest {
  string message = 1;
  string context = 2; // relationship type, previous session info
  string session_id = 3;
}

message ChatResponse {
  string response = 1;
}
```

#### Streaming AI Chat
```protobuf
message ChatMessage {
  string message = 1;
  string context = 2;
}
```

### 7. Rating Service

#### Create Rating
```protobuf
message CreateRatingRequest {
  string session_id = 1;
  int32 rating = 2; // 1-5
  string review_text = 3;
}

message CreateRatingResponse {
  Rating rating = 1;
}
```

#### Get Advisor Ratings
```protobuf
message GetAdvisorRatingsRequest {
  string advisor_id = 1;
  int32 limit = 2;
  int32 offset = 3;
}

message GetAdvisorRatingsResponse {
  repeated Rating ratings = 1;
}
```

### 8. Admin Service

#### Get Pending Advisors
```protobuf
message GetPendingAdvisorsRequest {
  int32 limit = 1;
  int32 offset = 2;
}

message GetPendingAdvisorsResponse {
  repeated common.Advisor advisors = 1;
}
```

#### Approve Advisor
```protobuf
message ApproveAdvisorRequest {
  string advisor_id = 1;
}

message ApproveAdvisorResponse {
  bool success = 1;
}
```

#### Get Flags
```protobuf
message GetFlagsRequest {
  int32 limit = 1;
  int32 offset = 2;
}

message GetFlagsResponse {
  repeated AdminFlag flags = 1;
}
```

#### Block User
```protobuf
message BlockUserRequest {
  string user_id = 1;
}

message BlockUserResponse {
  bool success = 1;
}
```

## Data Models

### User
```protobuf
message User {
  string id = 1;
  string email = 2;
  string phone = 3;
  string display_name = 4;
  Role role = 5; // USER, ADVISOR, ADMIN
  Gender gender = 6;
  string dob = 7;
  string created_at = 8;
  string updated_at = 9;
  bool is_active = 10;
}
```

### Advisor
```protobuf
message Advisor {
  string id = 1;
  string user_id = 2;
  string bio = 3;
  int32 experience_years = 4;
  repeated string languages = 5;
  repeated string specializations = 6;
  bool is_verified = 7;
  double hourly_rate = 8;
  AdvisorStatus status = 9; // ONLINE, OFFLINE, BUSY, PENDING
  string created_at = 10;
  string updated_at = 11;
}
```

### Session
```protobuf
message Session {
  string id = 1;
  string user_id = 2;
  string advisor_id = 3;
  SessionType type = 4; // CHAT, CALL, AI_CHAT
  string started_at = 5;
  string ended_at = 6;
  SessionStatus status = 7; // ONGOING, ENDED, CANCELLED
}
```

### ChatMessage
```protobuf
message ChatMessage {
  string id = 1;
  string session_id = 2;
  string sender_type = 3; // USER, ADVISOR, AI
  string sender_id = 4;
  string content = 5;
  string created_at = 6;
  bool is_read = 7;
}
```

### Rating
```protobuf
message Rating {
  string id = 1;
  string session_id = 2;
  string user_id = 3;
  string advisor_id = 4;
  int32 rating = 5;
  string review_text = 6;
  string created_at = 7;
}
```

## Error Handling

All API responses follow gRPC status codes:

- `OK (0)`: Success
- `InvalidArgument (3)`: Bad request parameters
- `Unauthenticated (16)`: Invalid or expired token
- `PermissionDenied (7)`: Insufficient permissions
- `NotFound (5)`: Resource not found
- `AlreadyExists (6)`: Resource already exists
- `ResourceExhausted (8)`: Rate limit exceeded
- `Internal (13)`: Internal server error

## Rate Limiting

- **Auth endpoints**: 5 requests per minute
- **Chat endpoints**: 30 requests per minute  
- **AI endpoints**: 10 requests per minute
- **General endpoints**: 60 requests per minute

## Health Check

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2023-12-01T10:00:00Z"
}
```

## WebSocket Events

### Connection
Connect to WebSocket with session_id and token query parameters.

### Events
- `MESSAGE`: Chat message
- `TYPING`: User is typing indicator
- `USER_JOINED`: User joined session
- `USER_LEFT`: User left session

### Message Types

#### Outgoing (Client → Server)
```json
{
  "type": "MESSAGE",
  "content": "Your message here"
}
```

#### Incoming (Server → Client)
```json
{
  "type": "MESSAGE",
  "session_id": "uuid",
  "sender_id": "uuid",
  "content": "Message content",
  "timestamp": "2023-12-01T10:00:00Z"
}
```

## Configuration

See `.env.example` for all available configuration options.

## Development

### Running the Server
```bash
go run cmd/server/main.go
```

### Running with Docker
```bash
docker-compose up -d
```

### Database Migrations
```bash
go run cmd/migrate/main.go up
```

## Support

For API support, contact the development team or check the GitHub issues.