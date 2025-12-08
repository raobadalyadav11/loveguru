package auth

import (
	"context"
	"database/sql"

	"loveguru/internal/db"

	"github.com/google/uuid"
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) CreateUser(ctx context.Context, email, phone, passwordHash, displayName, role string) (db.User, error) {
	return r.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        sql.NullString{String: email, Valid: email != ""},
		Phone:        sql.NullString{String: phone, Valid: phone != ""},
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Role:         role,
	})
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return r.queries.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
}

func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (db.User, error) {
	return r.queries.GetUserByPhone(ctx, sql.NullString{String: phone, Valid: true})
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (db.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return db.User{}, err
	}
	return r.queries.GetUserByID(ctx, uid)
}

func (r *Repository) UpdateUser(ctx context.Context, id, displayName string, gender sql.NullString, dob sql.NullTime) (db.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return db.User{}, err
	}
	return r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:          uid,
		DisplayName: displayName,
		Gender:      gender,
		Dob:         dob,
	})
}
