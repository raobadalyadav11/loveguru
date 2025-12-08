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

// publicMethods is a set of gRPC methods that don't require authentication.
// Using a map provides O(1) lookup time, which is more efficient than iterating a slice.
var publicMethods = map[string]struct{}{
	"/loveguru.auth.AuthService/Register": {},
	"/loveguru.auth.AuthService/Login":    {},
	"/loveguru.auth.AuthService/Refresh":  {},
}

func isPublicMethod(method string) bool {
	_, ok := publicMethods[method]
	return ok
}

func UnaryAuthInterceptor(jwtSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Allow public methods that don't require authentication
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// For protected methods, require authentication
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
		// Allow public methods that don't require authentication
		if isPublicMethod(info.FullMethod) {
			return handler(srv, stream)
		}

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

	// Check for empty metadata keys
	for key := range md {
		if key == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid metadata key")
		}
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	tokenString := authHeader[0]
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	} else {
		return nil, status.Error(codes.Unauthenticated, "authorization header must start with 'Bearer '")
	}

	if tokenString == "" {
		return nil, status.Error(codes.Unauthenticated, "empty token")
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
