package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"loveguru/internal/config"

	"github.com/google/uuid"
)

// TestDatabase provides a test database setup
type TestDatabase struct {
	DB      *sql.DB
	Queries *Queries
	Ctx     context.Context
}

// SetupTestDatabase creates a test database connection
func SetupTestDatabase() (*TestDatabase, error) {
	// Load test configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Use test database or create a new one
	dbName := cfg.Database.DBName + "_test"

	// Connect to postgres to create test database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.SSLMode)

	tempDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer tempDB.Close()

	// Create test database if it doesn't exist
	_, err = tempDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		// Database might already exist, which is fine
		log.Printf("Warning: Could not create test database: %v", err)
	}

	// Connect to test database
	cfg.Database.DBName = dbName
	testDB, err := NewDB(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Run migrations
	if err := runTestMigrations(testDB); err != nil {
		testDB.Close()
		return nil, fmt.Errorf("failed to run test migrations: %w", err)
	}

	return &TestDatabase{
		DB:      testDB,
		Queries: New(testDB),
		Ctx:     context.Background(),
	}, nil
}

// TeardownTestDatabase cleans up the test database
func (tdb *TestDatabase) TeardownTestDatabase() {
	if tdb.DB != nil {
		tdb.DB.Close()
	}
}

// runTestMigrations runs the database schema migrations
func runTestMigrations(db *sql.DB) error {
	migrations := []string{
		// Create extensions
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,

		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email TEXT,
			phone TEXT,
			password_hash TEXT NOT NULL,
			display_name TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('USER', 'ADVISOR', 'ADMIN')),
			gender TEXT CHECK (gender IN ('MALE', 'FEMALE', 'OTHER')),
			dob DATE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			is_active BOOLEAN DEFAULT TRUE,
			UNIQUE(email),
			UNIQUE(phone),
		 CHECK (email IS NOT NULL OR phone IS NOT NULL)
		);`,

		// Advisors table
		`CREATE TABLE IF NOT EXISTS advisors (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			bio TEXT,
			experience_years INTEGER,
			languages TEXT[],
			specializations TEXT[],
			is_verified BOOLEAN DEFAULT FALSE,
			hourly_rate DECIMAL(10,2),
			status TEXT DEFAULT 'PENDING' CHECK (status IN ('ONLINE', 'OFFLINE', 'BUSY', 'PENDING')),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id)
		);`,

		// Sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id),
			advisor_id UUID REFERENCES users(id),
			type TEXT NOT NULL CHECK (type IN ('CHAT', 'CALL', 'AI_CHAT')),
			started_at TIMESTAMPTZ DEFAULT NOW(),
			ended_at TIMESTAMPTZ,
			status TEXT DEFAULT 'ONGOING' CHECK (status IN ('ONGOING', 'ENDED', 'CANCELLED'))
		);`,

		// Chat messages table
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			sender_type TEXT NOT NULL CHECK (sender_type IN ('USER', 'ADVISOR', 'AI')),
			sender_id UUID NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			is_read BOOLEAN DEFAULT FALSE
		);`,

		// Call logs table
		`CREATE TABLE IF NOT EXISTS call_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			external_call_id TEXT,
			started_at TIMESTAMPTZ,
			ended_at TIMESTAMPTZ,
			duration_seconds INTEGER,
			status TEXT
		);`,

		// Ratings table
		`CREATE TABLE IF NOT EXISTS ratings (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			session_id UUID NOT NULL REFERENCES sessions(id),
			user_id UUID NOT NULL REFERENCES users(id),
			advisor_id UUID NOT NULL REFERENCES users(id),
			rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
			review_text TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		// AI interactions table
		`CREATE TABLE IF NOT EXISTS ai_interactions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id),
			prompt TEXT NOT NULL,
			response TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		// Admin flags table
		`CREATE TABLE IF NOT EXISTS admin_flags (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			reported_by UUID NOT NULL REFERENCES users(id),
			reported_user_id UUID REFERENCES users(id),
			reported_advisor_id UUID REFERENCES users(id),
			reason TEXT NOT NULL,
			session_id UUID REFERENCES sessions(id),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			status TEXT DEFAULT 'PENDING'
		);`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// TestUserCRUD tests basic user CRUD operations
func TestUserCRUD(t *testing.T) {
	tdb, err := SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer tdb.TeardownTestDatabase()

	// Test Create User
	user, err := tdb.Queries.CreateUser(tdb.Ctx, CreateUserParams{
		Email:        sql.NullString{String: "test@example.com", Valid: true},
		Phone:        sql.NullString{Valid: false},
		PasswordHash: "hashed_password",
		DisplayName:  "Test User",
		Role:         "USER",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.DisplayName != "Test User" {
		t.Errorf("Expected display name 'Test User', got '%s'", user.DisplayName)
	}

	// Test Get User by ID
	retrievedUser, err := tdb.Queries.GetUserByID(tdb.Ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
	}

	// Test Update User
	updatedUser, err := tdb.Queries.UpdateUser(tdb.Ctx, UpdateUserParams{
		ID:          user.ID,
		DisplayName: "Updated Test User",
		Gender:      sql.NullString{String: "MALE", Valid: true},
		Dob:         sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	if updatedUser.DisplayName != "Updated Test User" {
		t.Errorf("Expected updated display name 'Updated Test User', got '%s'", updatedUser.DisplayName)
	}
}

// TestAdvisorCRUD tests advisor CRUD operations
func TestAdvisorCRUD(t *testing.T) {
	tdb, err := SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer tdb.TeardownTestDatabase()

	// Create a user first
	user, err := tdb.Queries.CreateUser(tdb.Ctx, CreateUserParams{
		Email:        sql.NullString{String: "advisor@example.com", Valid: true},
		Phone:        sql.NullString{Valid: false},
		PasswordHash: "hashed_password",
		DisplayName:  "Test Advisor",
		Role:         "ADVISOR",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test Create Advisor
	advisor, err := tdb.Queries.CreateAdvisor(tdb.Ctx, CreateAdvisorParams{
		UserID:          user.ID,
		Bio:             sql.NullString{String: "Experienced advisor", Valid: true},
		ExperienceYears: sql.NullInt32{Int32: 5, Valid: true},
		Languages:       []string{"English", "Spanish"},
		Specializations: []string{"Love", "Relationships"},
		HourlyRate:      sql.NullString{String: "50.00", Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to create advisor: %v", err)
	}

	if advisor.Bio.String != "Experienced advisor" {
		t.Errorf("Expected bio 'Experienced advisor', got '%s'", advisor.Bio.String)
	}

	// Test Get Advisor by User ID
	retrievedAdvisor, err := tdb.Queries.GetAdvisorByUserID(tdb.Ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get advisor by user ID: %v", err)
	}

	if retrievedAdvisor.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedAdvisor.UserID)
	}
}

// TestSessionCRUD tests session CRUD operations
func TestSessionCRUD(t *testing.T) {
	tdb, err := SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer tdb.TeardownTestDatabase()

	// Create users
	user1, err := tdb.Queries.CreateUser(tdb.Ctx, CreateUserParams{
		Email:        sql.NullString{String: "user1@example.com", Valid: true},
		Phone:        sql.NullString{Valid: false},
		PasswordHash: "hashed_password",
		DisplayName:  "User 1",
		Role:         "USER",
	})
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2, err := tdb.Queries.CreateUser(tdb.Ctx, CreateUserParams{
		Email:        sql.NullString{String: "user2@example.com", Valid: true},
		Phone:        sql.NullString{Valid: false},
		PasswordHash: "hashed_password",
		DisplayName:  "User 2",
		Role:         "ADVISOR",
	})
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Test Create Session
	session, err := tdb.Queries.CreateSession(tdb.Ctx, CreateSessionParams{
		UserID:    user1.ID,
		AdvisorID: uuid.NullUUID{UUID: user2.ID, Valid: true},
		Type:      "CHAT",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.Type != "CHAT" {
		t.Errorf("Expected session type 'CHAT', got '%s'", session.Type)
	}

	// Test Get Session by ID
	retrievedSession, err := tdb.Queries.GetSessionByID(tdb.Ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session by ID: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrievedSession.ID)
	}

	// Test Update Session Status
	err = tdb.Queries.UpdateSessionStatus(tdb.Ctx, UpdateSessionStatusParams{
		ID:     session.ID,
		Status: sql.NullString{String: "ENDED", Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to update session status: %v", err)
	}

	// Verify status was updated
	updatedSession, err := tdb.Queries.GetSessionByID(tdb.Ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	if updatedSession.Status.String != "ENDED" {
		t.Errorf("Expected session status 'ENDED', got '%s'", updatedSession.Status.String)
	}
}

// TestForeignKeyConstraints tests foreign key constraint violations
func TestForeignKeyConstraints(t *testing.T) {
	tdb, err := SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer tdb.TeardownTestDatabase()

	// Test creating a session with non-existent user
	nonExistentID := uuid.New()
	_, err = tdb.Queries.CreateSession(tdb.Ctx, CreateSessionParams{
		UserID:    nonExistentID,
		AdvisorID: uuid.NullUUID{UUID: nonExistentID, Valid: true},
		Type:      "CHAT",
	})

	if err == nil {
		t.Error("Expected error when creating session with non-existent user, but got none")
	}

	// Test creating a chat message with non-existent session
	_, err = tdb.Queries.InsertMessage(tdb.Ctx, InsertMessageParams{
		SessionID:  nonExistentID,
		SenderType: "USER",
		SenderID:   nonExistentID,
		Content:    "Test message",
	})

	if err == nil {
		t.Error("Expected error when inserting message with non-existent session, but got none")
	}
}

// TestTransaction tests database transactions
func TestTransaction(t *testing.T) {
	tdb, err := SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer tdb.TeardownTestDatabase()

	// Test successful transaction
	err = Transaction(tdb.Ctx, tdb.DB, func(queries *Queries) error {
		user, err := queries.CreateUser(tdb.Ctx, CreateUserParams{
			Email:        sql.NullString{String: "transaction@example.com", Valid: true},
			Phone:        sql.NullString{Valid: false},
			PasswordHash: "hashed_password",
			DisplayName:  "Transaction User",
			Role:         "USER",
		})
		if err != nil {
			return err
		}

		// This should succeed
		_, err = queries.CreateSession(tdb.Ctx, CreateSessionParams{
			UserID:    user.ID,
			AdvisorID: uuid.NullUUID{Valid: false},
			Type:      "AI_CHAT",
		})
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Test failed transaction (should rollback)
	nonExistentID := uuid.New()
	err = Transaction(tdb.Ctx, tdb.DB, func(queries *Queries) error {
		user, err := queries.CreateUser(tdb.Ctx, CreateUserParams{
			Email:        sql.NullString{String: "rollback@example.com", Valid: true},
			Phone:        sql.NullString{Valid: false},
			PasswordHash: "hashed_password",
			DisplayName:  "Rollback User",
			Role:         "USER",
		})
		if err != nil {
			return err
		}

		// This should fail due to foreign key constraint
		_, err = queries.CreateSession(tdb.Ctx, CreateSessionParams{
			UserID:    user.ID,
			AdvisorID: uuid.NullUUID{UUID: nonExistentID, Valid: true},
			Type:      "CHAT",
		})
		return err
	})

	if err == nil {
		t.Error("Expected transaction to fail due to foreign key constraint, but it succeeded")
	}

	// Verify no partial data was created
	_, err = tdb.Queries.GetUserByEmail(tdb.Ctx, sql.NullString{String: "rollback@example.com", Valid: true})
	if err == nil {
		t.Error("Expected no user to be created after failed transaction, but found one")
	}
}

// RunAllTests runs all database tests
func RunAllTests(t *testing.T) {
	t.Run("UserCRUD", TestUserCRUD)
	t.Run("AdvisorCRUD", TestAdvisorCRUD)
	t.Run("SessionCRUD", TestSessionCRUD)
	t.Run("ForeignKeyConstraints", TestForeignKeyConstraints)
	t.Run("Transaction", TestTransaction)
}

// Example usage function for manual testing
func ExampleManualTest() {
	tdb, err := SetupTestDatabase()
	if err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}
	defer tdb.TeardownTestDatabase()

	fmt.Println("Database test setup completed successfully!")
	fmt.Printf("Database connection stats: %+v\n", tdb.DB.Stats())
}
