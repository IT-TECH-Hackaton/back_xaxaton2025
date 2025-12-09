package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golos/internal/api/handlers"
	"golos/internal/config"
	"golos/internal/repository"
	"golos/internal/service/audio"
	"golos/internal/service/auth"
	"golos/internal/service/email"
	"golos/internal/service/gigachat"
	"golos/internal/storage"
)

type Server struct {
	config         *config.Config
	httpServer     *http.Server
	gigaChat       *gigachat.Client
	audioClient    *audio.Client
	sessionManager *storage.SessionManager
	authHandler    *handlers.AuthHandler
}

func NewServer(cfg *config.Config) *Server {
	sessionTTL := 30 * time.Minute
	if cfg.Server.SessionTTL > 0 {
		sessionTTL = cfg.Server.SessionTTL
	}

	userRepo := repository.NewUserRepository()
	emailVerificationRepo := repository.NewEmailVerificationRepository()
	passwordResetRepo := repository.NewPasswordResetRepository()
	jwtService := auth.NewJWTService(&cfg.JWT)
	passwordService := auth.NewPasswordService()
	emailService := email.NewEmailService(
		cfg.Email.Host,
		cfg.Email.Port,
		cfg.Email.User,
		cfg.Email.Password,
		cfg.Email.From,
	)
	authService := auth.NewAuthService(
		userRepo,
		emailVerificationRepo,
		passwordResetRepo,
		jwtService,
		passwordService,
		emailService,
	)
	authHandler := handlers.NewAuthHandler(authService)

	return &Server{
		config:         cfg,
		gigaChat:       gigachat.NewClient(cfg.GigaChat),
		audioClient:    audio.NewClient(cfg.AudioService.URL),
		sessionManager: storage.NewSessionManager(sessionTTL),
		authHandler:    authHandler,
	}
}

func (s *Server) Start() error {
	router := s.setupRouter()

	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Сервер запущен на %s\n", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.sessionManager.Stop()
	return s.httpServer.Shutdown(ctx)
}
