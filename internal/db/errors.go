package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// DatabaseError provides detailed database error information
type DatabaseError struct {
	Operation string
	Table     string
	Err       error
}

func (e *DatabaseError) Error() string {
	return fmt.Errorf("%s on table %s failed: %w", e.Operation, e.Table, e.Err).Error()
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// IsNotFound checks if the error is a "not found" error
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// IsDuplicateKey checks if the error is a duplicate key violation
func IsDuplicateKey(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// IsForeignKeyViolation checks if the error is a foreign key violation
func IsForeignKeyViolation(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503" // foreign_key_violation
	}
	return false
}

// IsConstraintViolation checks if the error is a constraint violation
func IsConstraintViolation(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23502" || pgErr.Code == "23505" || pgErr.Code == "23503"
	}
	return false
}

// EnhancedDBError wraps database errors with additional context
func EnhancedDBError(operation, table string, err error) *DatabaseError {
	if err == nil {
		return nil
	}
	return &DatabaseError{
		Operation: operation,
		Table:     table,
		Err:       err,
	}
}

// RetryWithBackoff retries a database operation with exponential backoff
func RetryWithBackoff(ctx context.Context, maxRetries int, op func() error) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms, etc.
			backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err = op()
		if err == nil {
			return nil
		}

		// Don't retry on client errors or constraint violations
		if IsNotFound(err) || IsConstraintViolation(err) {
			return err
		}

		// For transient errors (connection issues, timeouts, etc.), continue retrying
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			// Retry on connection errors, timeouts, and temporary issues
			if pgErr.Code == "08003" || pgErr.Code == "08006" || pgErr.Code == "08001" || pgErr.Code == "08004" {
				continue
			}
		}

		// Don't retry other types of errors
		break
	}

	return err
}

// CheckConnection checks if the database connection is healthy
func CheckConnection(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

// GetConnectionStats returns database connection statistics
func GetConnectionStats(db *sql.DB) sql.DBStats {
	return db.Stats()
}
