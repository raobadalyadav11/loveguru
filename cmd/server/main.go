package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loveguru/internal/admin"
	"loveguru/internal/advisor"
	"loveguru/internal/ai"
	"loveguru/internal/auth"
	"loveguru/internal/cache"
	"loveguru/internal/call"
	"loveguru/internal/chat"
	"loveguru/internal/config"
	"loveguru/internal/db"
	"loveguru/internal/grpc/middleware"
	"loveguru/internal/logger"
	"loveguru/internal/notifications"
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

	// Initialize logger
	_ = logger.NewLogger()

	// Connect to database
	dbConn, err := db.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Create queries instance
	queries := db.New(dbConn)

	// Initialize Redis cache (optional)
	var cacheService *cache.Cache
	if cfg.Redis.Host != "" {
		cacheService = cache.NewCache(
			cfg.Redis.Host+":"+string(rune(cfg.Redis.Port)),
			cfg.Redis.Password,
			cfg.Redis.DB,
		)
		defer cacheService.Close()
	}

	// Initialize notification service
	_ = notifications.NewNotificationService()

	// Create services
	authService := auth.NewService(auth.NewRepository(queries), cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	userService := user.NewService(queries)
	advisorService := advisor.NewService(queries)

	// Create WebSocket hub for real-time chat
	chatHub := chat.NewHub(chat.NewService(queries))
	go chatHub.Run()

	chatService := chat.NewService(queries)
	callService := call.NewService(queries)
	ratingService := rating.NewService(queries)

	// Initialize AI service with real OpenAI integration
	aiService := ai.NewService(queries, os.Getenv("OPENAI_API_KEY"), "https://api.openai.com")
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

	// Initialize rate limiter
	_ = middleware.NewRateLimiter()

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

	// Setup HTTP server for WebSocket connections
	mux := http.NewServeMux()

	// WebSocket handler for chat
	mux.HandleFunc("/ws/chat", func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session_id")
		token := r.URL.Query().Get("token")

		if sessionID == "" || token == "" {
			http.Error(w, "Missing session_id or token", http.StatusBadRequest)
			return
		}

		// Validate token and extract user ID (simplified)
		userID := "user123" // In real implementation, validate JWT token

		chatHub.HandleWebSocket(w, r, sessionID, userID)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
	})

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start servers in goroutines
	grpcServer := make(chan bool)
	httpServerChan := make(chan bool)

	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("gRPC server listening at %v", lis.Addr())
		grpcServer <- true
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	go func() {
		log.Printf("HTTP server listening at :8080")
		httpServerChan <- true
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down servers...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.GracefulStop()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	log.Println("servers stopped")
}
