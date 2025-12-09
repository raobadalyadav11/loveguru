package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ValidationError represents a data validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}
	if len(email) > 254 {
		return &ValidationError{Field: "email", Message: "email too long"}
	}
	// Basic email validation - in production, use a more robust validation
	if email[0] == '@' || email[len(email)-1] == '@' {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}
	return nil
}

// ValidatePhone validates phone number format
func ValidatePhone(phone string) error {
	if phone == "" {
		return nil // Phone is optional
	}
	if len(phone) < 10 || len(phone) > 15 {
		return &ValidationError{Field: "phone", Message: "phone number must be 10-15 digits"}
	}
	return nil
}

// ValidateDisplayName validates display name
func ValidateDisplayName(name string) error {
	if name == "" {
		return &ValidationError{Field: "display_name", Message: "display name is required"}
	}
	if len(name) > 100 {
		return &ValidationError{Field: "display_name", Message: "display name too long"}
	}
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return &ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}
	return nil
}

// ValidateUUID validates UUID format
func ValidateUUID(id string) error {
	_, err := uuid.Parse(id)
	if err != nil {
		return &ValidationError{Field: "id", Message: "invalid UUID format"}
	}
	return nil
}

// ValidateRating validates rating value
func ValidateRating(rating int32) error {
	if rating < 1 || rating > 5 {
		return &ValidationError{Field: "rating", Message: "rating must be between 1 and 5"}
	}
	return nil
}

// ValidateExperienceYears validates experience years
func ValidateExperienceYears(years int32) error {
	if years < 0 {
		return &ValidationError{Field: "experience_years", Message: "experience years cannot be negative"}
	}
	if years > 50 {
		return &ValidationError{Field: "experience_years", Message: "experience years too high"}
	}
	return nil
}

// ValidateHourlyRate validates hourly rate
func ValidateHourlyRate(rate float64) error {
	if rate < 0 {
		return &ValidationError{Field: "hourly_rate", Message: "hourly rate cannot be negative"}
	}
	if rate > 1000 {
		return &ValidationError{Field: "hourly_rate", Message: "hourly rate too high"}
	}
	return nil
}

// ValidateLanguages validates language array
func ValidateLanguages(languages []string) error {
	if len(languages) > 20 {
		return &ValidationError{Field: "languages", Message: "too many languages"}
	}
	for i, lang := range languages {
		if len(lang) > 50 {
			return &ValidationError{Field: fmt.Sprintf("languages[%d]", i), Message: "language name too long"}
		}
	}
	return nil
}

// ValidateSpecializations validates specializations array
func ValidateSpecializations(specializations []string) error {
	if len(specializations) > 20 {
		return &ValidationError{Field: "specializations", Message: "too many specializations"}
	}
	for i, spec := range specializations {
		if len(spec) > 100 {
			return &ValidationError{Field: fmt.Sprintf("specializations[%d]", i), Message: "specialization name too long"}
		}
	}
	return nil
}

// ValidateSessionType validates session type
func ValidateSessionType(sessionType string) error {
	validTypes := map[string]bool{
		"CHAT":    true,
		"CALL":    true,
		"AI_CHAT": true,
	}
	if !validTypes[sessionType] {
		return &ValidationError{Field: "type", Message: "invalid session type"}
	}
	return nil
}

// ValidateSessionStatus validates session status
func ValidateSessionStatus(status string) error {
	validStatuses := map[string]bool{
		"ONGOING":   true,
		"ENDED":     true,
		"CANCELLED": true,
	}
	if !validStatuses[status] {
		return &ValidationError{Field: "status", Message: "invalid session status"}
	}
	return nil
}

// ValidateUserRole validates user role
func ValidateUserRole(role string) error {
	validRoles := map[string]bool{
		"USER":    true,
		"ADVISOR": true,
		"ADMIN":   true,
	}
	if !validRoles[role] {
		return &ValidationError{Field: "role", Message: "invalid user role"}
	}
	return nil
}

// ValidateAdvisorStatus validates advisor status
func ValidateAdvisorStatus(status string) error {
	validStatuses := map[string]bool{
		"ONLINE":  true,
		"OFFLINE": true,
		"BUSY":    true,
		"PENDING": true,
	}
	if !validStatuses[status] {
		return &ValidationError{Field: "status", Message: "invalid advisor status"}
	}
	return nil
}

// SafeNullString creates a safe sql.NullString
func SafeNullString(value string) sql.NullString {
	return sql.NullString{
		String: value,
		Valid:  value != "",
	}
}

// SafeNullTime creates a safe sql.NullTime
func SafeNullTime(value time.Time) sql.NullString {
	return sql.NullString{
		String: value.Format("2006-01-02"),
		Valid:  !value.IsZero(),
	}
}

// SafeNullInt32 creates a safe sql.NullInt32
func SafeNullInt32(value int32, validIfPositive bool) sql.NullInt32 {
	return sql.NullInt32{
		Int32: value,
		Valid: validIfPositive && value != 0,
	}
}

// ValidateRequiredFields validates required fields for user creation
func ValidateRequiredUserFields(email, phone, password, displayName, role string) error {
	if email == "" && phone == "" {
		return errors.New("either email or phone is required")
	}

	if err := ValidateEmail(email); err != nil {
		return err
	}

	if err := ValidatePhone(phone); err != nil {
		return err
	}

	if err := ValidatePassword(password); err != nil {
		return err
	}

	if err := ValidateDisplayName(displayName); err != nil {
		return err
	}

	if err := ValidateUserRole(role); err != nil {
		return err
	}

	return nil
}

// ValidateSessionData validates session-related data
func ValidateSessionData(userID, advisorID uuid.UUID, sessionType string) error {
	if err := ValidateUUID(userID.String()); err != nil {
		return err
	}

	if err := ValidateSessionType(sessionType); err != nil {
		return err
	}

	// AdvisorID can be null for AI_CHAT sessions
	if advisorID != (uuid.UUID{}) {
		if err := ValidateUUID(advisorID.String()); err != nil {
			return err
		}
	}

	return nil
}
