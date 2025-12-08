package main

import (
	"log"
	"net"

	"loveguru/internal/admin"
	"loveguru/internal/advisor"
	"loveguru/internal/ai"
	"loveguru/internal/auth"
	"loveguru/internal/call"
	"loveguru/internal/chat"
	"loveguru/internal/config"
	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/internal/rating"
	"loveguru/internal/user"

	pbadmin "loveguru/proto/admin"
	pbadvisor "loveguru/proto/advisor"
	pbai "loveguru/proto/ai"
	pbauth "loveguru/proto/auth"
	pbcall "loveguru/proto/call"
	pbchat "loveguru/proto/chat"
	pbrating "loveguru/proto/rating"
	pbuser "loveguru/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to database
	dbConn, err := db.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Create queries instance
	queries := db.New(dbConn)

	// Create services
	authService := auth.NewService(auth.NewRepository(queries), cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	userService := user.NewService(queries)
	advisorService := advisor.NewService(queries)
	chatService := chat.NewService(queries)
	callService := call.NewService(queries)
	ratingService := rating.NewService(queries)
	aiService := ai.NewService(queries, "dummy-key", "https://api.openai.com")
	adminService := admin.NewService(queries)

	// Create handlers
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	advisorHandler := advisor.NewHandler(advisorService)
	chatHandler := chat.NewHandler(chatService)
	callHandler := call.NewHandler(callService)
	ratingHandler := rating.NewHandler(ratingService)
	aiHandler := ai.NewHandler(aiService)
	adminHandler := admin.NewHandler(adminService)

	// Create gRPC server with interceptors
	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.UnaryAuthInterceptor(cfg.JWT.Secret)),
		grpc.StreamInterceptor(middleware.StreamAuthInterceptor(cfg.JWT.Secret)),
	)

	// Register services
	pbauth.RegisterAuthServiceServer(s, authHandler)
	pbuser.RegisterUserServiceServer(s, userHandler)
	pbadvisor.RegisterAdvisorServiceServer(s, advisorHandler)
	pbchat.RegisterChatServiceServer(s, chatHandler)
	pbcall.RegisterCallServiceServer(s, callHandler)
	pbrating.RegisterRatingServiceServer(s, ratingHandler)
	pbai.RegisterAIServiceServer(s, aiHandler)
	pbadmin.RegisterAdminServiceServer(s, adminHandler)

	reflection.Register(s)

	// Start server
	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
