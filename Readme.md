App: Love advice / love therapy platform
Backend Language: Go (Golang)
Architecture Style: Modular monolith (Phase 1) → can evolve to microservices later
Main responsibilities:

Auth & user management (users + advisers)

Adviser marketplace & rating system

Real‑time chat

Call session management (via external VoIP provider)

AI assistant integration

Admin panel APIs

2. Tech Stack (Backend)
Language: Go 1.22+

Framework:

HTTP router: Gin (or Fiber, pick one and stick to it)

DB:

Primary: PostgreSQL (users, advisers, ratings, sessions, calls)

Secondary: Redis (caching, WebSocket presence, rate limiting)

Realtime:

WebSockets (self-hosted) or 3rd party (Pusher/Ably) – here we assume self‑hosted WS in Go

Message Queue (optional later): NATS / RabbitMQ (for notifications, async jobs)

AI Integration: HTTP client to LLM API (OpenAI etc.)

Config: viper or env variables

Migrations: golang-migrate

Auth: JWT with refresh tokens

Deployment Target: Docker + Kubernetes or simple Docker + ECS/GCE

3. High-Level Architecture (Backend)
3.1 Logical Components
API Gateway / HTTP Server

Receives all requests

Authentication / authorization middleware

Routes to modules

Modules / Domains

Auth Module

User Module

Advisor Module

Chat Module

Call Module

AI Assistant Module

Rating & Review Module

Admin Module

Infra Layer

DB access (Postgres)

Cache (Redis)

External integrations:

VoIP provider (Twilio/Agora/etc.)

AI provider (OpenAI, etc.)

Notification service (FCM, APNS)

4. Go Project Structure (Example)
love-advice-backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   ├── http/
│   │   ├── middleware/
│   │   └── router.go
│   ├── auth/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── user/
│   ├── advisor/
│   ├── chat/
│   │   ├── ws_hub.go
│   │   ├── handler.go
│   │   └── service.go
│   ├── call/
│   ├── rating/
│   ├── ai/
│   ├── admin/
│   ├── models/
│   ├── db/
│   │   ├── postgres.go
│   │   └── migrations/
│   ├── cache/
│   ├── utils/
│   └── logger/
├── pkg/
│   └── (optional shared libs)
├── go.mod
└── go.sum
5. Core Entities / Data Model
5.1 Tables (Main)
users

id (uuid, pk)

email (nullable if phone only)

phone (nullable if email only)

password_hash

display_name

role (USER / ADVISOR / ADMIN)

gender (optional)

dob (optional)

created_at, updated_at

is_active

advisors

id (uuid, pk)

user_id (fk -> users)

bio

experience_years

languages (array/jsonb)

specializations (jsonb: breakup, marriage, dating, etc.)

is_verified (KYC done)

hourly_rate or per_min_rate (for future paid model)

status (ONLINE, OFFLINE, BUSY)

created_at, updated_at

sessions (chat or call sessions)

id (uuid, pk)

user_id

advisor_id

type (CHAT, CALL, AI_CHAT)

started_at

ended_at

status (ONGOING, ENDED, CANCELLED)

chat_messages

id (uuid, pk)

session_id (fk -> sessions)

sender_type (USER, ADVISOR, AI)

sender_id

content (text)

created_at

is_read

call_logs

id (uuid, pk)

session_id (fk -> sessions)

external_call_id (id from VoIP provider)

started_at

ended_at

duration_seconds

status

ratings

id (uuid, pk)

session_id

user_id

advisor_id

rating (1–5)

review_text

created_at

ai_interactions

id (uuid, pk)

user_id

prompt

response

created_at

admin_flags

id

reported_by

reported_user_id or advisor_id

reason

session_id (optional)

created_at

status

(For now I’m skipping payments tables since you said initially free.)

6. API Design (Key Endpoints)
I’ll keep it REST-style; actual paths can be adjusted.

6.1 Auth
POST /api/v1/auth/register

Input: email/phone, password, display_name, role

Output: user info + access_token + refresh_token

POST /api/v1/auth/login

Input: email+password or phone+otp

Output: tokens

POST /api/v1/auth/refresh

POST /api/v1/auth/logout

6.2 User
GET /api/v1/users/me

PATCH /api/v1/users/me

GET /api/v1/users/me/sessions

6.3 Advisor
GET /api/v1/advisors

Query params: rating_min, language, specialization, sort=top_rated|price|experience

GET /api/v1/advisors/{id}

POST /api/v1/advisors/apply (for advisers to create profile)

PATCH /api/v1/advisors/me (update bio, specialization, status)

6.4 Chat
POST /api/v1/sessions/chat

Create chat session (user ↔ advisor OR user ↔ AI)

GET /api/v1/sessions/{id}/messages

WebSocket endpoint:

GET /ws/chat?session_id=...&token=...

WebSocket message format example (JSON):

{
  "type": "MESSAGE",
  "session_id": "uuid",
  "sender_type": "USER",
  "content": "Hey, I need advice..."
}
Server broadcasts messages to the other participant (adviser or AI service).

6.5 Call
POST /api/v1/sessions/call

Create call session for user ↔ adviser

Backend calls VoIP API to create/join room

POST /api/v1/calls/{session_id}/end

GET /api/v1/calls/{session_id}

6.6 Ratings
POST /api/v1/sessions/{id}/rating

GET /api/v1/advisors/{id}/ratings

6.7 AI Assistant
POST /api/v1/ai/chat

Input: user message + context (relationship type, previous conversation id)

Output: AI message

Internally also used by WebSocket chat when sender_type == AI.

6.8 Admin
GET /api/v1/admin/advisors/pending

POST /api/v1/admin/advisors/{id}/approve

GET /api/v1/admin/flags

POST /api/v1/admin/users/{id}/block

7. Key Flows (How Things Work)
7.1 User Signup & Onboarding
App → POST /auth/register

Backend:

Validate data

Create row in users

If role = ADVISOR → create empty row in advisors (status: PENDING)

Return JWT tokens

App stores tokens securely and moves to home screen.

7.2 Adviser Discovery
App → GET /advisors?rating_min=4&language=en&sort=top_rated

Backend:

Query advisors + JOIN avg rating from ratings

Apply filters, sort

Return paginated list

7.3 Start Chat with Adviser
App → POST /sessions/chat with advisor_id

Backend:

Create session row with type=CHAT, status=ONGOING

Return session_id

App connects to ws/chat?session_id=...

Chat messages flow over WebSocket and are persisted in chat_messages.

7.4 AI Assistant Flow
User sends message → app calls:

Either WebSocket to chat service with sender_type=USER & session_type=AI

Or simple POST /ai/chat

Backend:

Stores message

Calls external AI (OpenAI) with sanitized prompt

Returns response & stores as sender_type=AI

7.5 Voice Call Flow (High Level)
App → POST /sessions/call with advisor_id

Backend:

Check adviser status = ONLINE

Create session row (type=CALL)

Call VoIP API → create room/token

Return call join token & room ID to both clients

Clients connect to VoIP SDK directly (voice goes via provider).

After call end:

Provider notifies backend via webhook OR client hits POST /calls/{session_id}/end

Backend updates call_logs, sessions table.

8. Cross-Cutting Concerns
8.1 Authentication & Authorization
Use JWT access token (short-lived) + refresh token (longer).

Middleware in Go to:

Verify token

Attach user_id & role to context

Restrict endpoints:

/admin/* → role == ADMIN

Adviser endpoints → role == ADVISOR

8.2 Validation
Use validation library (go-playground/validator) for request structs.

Standard error format:

{
  "error": "validation_error",
  "details": { "field": "email", "message": "invalid email" }
}
8.3 Logging
Structured logging using zerolog or logrus.

Log:

Request path, user_id, latency

Errors (DB, external API, etc.)

8.4 Config Management
Use env vars: DB URL, Redis URL, AI API key, VoIP keys.

Provide config.yml for local dev.

Load via viper.

8.5 Security
All endpoints over HTTPS.

Hash passwords using bcrypt.

Rate limiting on:

Auth endpoints

AI endpoints

Sanitize user input before passing to AI.

9. Future: Subscription & Paid Features (Hook Points)
Even though v1 is free, design with these in mind:

Add plans table (FREE, PRO).

Add user_subscriptions table.

Middleware that checks if user has subscription before:

Accessing top advisers

Starting extra sessions

Add advisor_rates table for per‑minute/per‑session pricing.

Payment integration module (Stripe/Razorpay).

10. What Devs Can Start With Right Now
Set up Go project structure (as above).

Implement:

Config loader

Postgres + Redis connection

Basic users, advisors models

Auth endpoints with JWT

Add adviser listing + filters.

Implement WebSocket chat hub for:

User ↔ Adviser messaging

User ↔ AI messaging (first just echo, then plug into real AI)

Add session creation & simple call logging (even before integrating VoIP).

