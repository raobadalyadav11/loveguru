package auth

import (
	"context"
	"database/sql"
	"errors"

	"loveguru/internal/db"
	"loveguru/internal/utils"
	"loveguru/proto/auth"
	"loveguru/proto/common"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo       *Repository
	jwtSecret  string
	accessTTL  int
	refreshTTL int
}

func NewService(repo *Repository, jwtSecret string, accessTTL, refreshTTL int) *Service {
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *Service) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	// Validate input
	if req.Email == "" && req.Phone == "" {
		return nil, errors.New("email or phone is required")
	}
	if req.Password == "" || req.DisplayName == "" {
		return nil, errors.New("password and display name are required")
	}

	// Check if user exists
	if req.Email != "" {
		_, err := s.repo.GetUserByEmail(ctx, req.Email)
		if err == nil {
			return nil, errors.New("email already exists")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if req.Phone != "" {
		_, err := s.repo.GetUserByPhone(ctx, req.Phone)
		if err == nil {
			return nil, errors.New("phone already exists")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user, err := s.repo.CreateUser(ctx, req.Email, req.Phone, string(hashed), req.DisplayName, req.Role.String())
	if err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(user.ID.String(), user.Role, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID.String(), s.jwtSecret, s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &auth.RegisterResponse{
		User: &common.User{
			Id:          user.ID.String(),
			Email:       user.Email.String,
			Phone:       user.Phone.String,
			DisplayName: user.DisplayName,
			Role:        common.Role(common.Role_value[user.Role]),
			Gender:      common.Gender(common.Gender_value[user.Gender.String]),
			Dob:         user.Dob.Time.Format("2006-01-02"),
			CreatedAt:   user.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   user.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
			IsActive:    user.IsActive.Bool,
		},
		Tokens: &common.Tokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *Service) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	var user db.User
	var err error

	if req.Email != "" {
		user, err = s.repo.GetUserByEmail(ctx, req.Email)
	} else if req.Phone != "" {
		user, err = s.repo.GetUserByPhone(ctx, req.Phone)
	} else {
		return nil, errors.New("email or phone is required")
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(user.ID.String(), user.Role, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID.String(), s.jwtSecret, s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &auth.LoginResponse{
		Tokens: &common.Tokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *Service) Refresh(ctx context.Context, req *auth.RefreshRequest) (*auth.RefreshResponse, error) {
	// Parse refresh token
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid user ID in token")
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Generate new tokens
	accessToken, err := utils.GenerateAccessToken(user.ID.String(), user.Role, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateRefreshToken(user.ID.String(), s.jwtSecret, s.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &auth.RefreshResponse{
		Tokens: &common.Tokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, req *auth.LogoutRequest) (*auth.LogoutResponse, error) {
	// In a real implementation, you might want to blacklist the token
	// For now, just return success
	return &auth.LogoutResponse{Success: true}, nil
}
