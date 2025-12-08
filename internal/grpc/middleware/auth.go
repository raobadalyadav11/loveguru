package middleware

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

type UserInfo struct {
	ID   string
	Role string
}

func UnaryAuthInterceptor(jwtSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		user, err := authenticate(ctx, jwtSecret)
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, UserContextKey, user)
		return handler(ctx, req)
	}
}

func StreamAuthInterceptor(jwtSecret string) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		user, err := authenticate(ctx, jwtSecret)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, UserContextKey, user)
		wrappedStream := &wrappedServerStream{ServerStream: stream, ctx: ctx}
		return handler(srv, wrappedStream)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func authenticate(ctx context.Context, jwtSecret string) (*UserInfo, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	if tokenString == authHeader[0] {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid claims")
	}

	return &UserInfo{ID: claims.UserID, Role: claims.Role}, nil
}

func GetUserFromContext(ctx context.Context) (*UserInfo, bool) {
	user, ok := ctx.Value(UserContextKey).(*UserInfo)
	return user, ok
}
