package auth

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"golos/internal/dto"
	"golos/internal/models"
	"golos/internal/repository"
	"golos/internal/service/email"
	"golos/internal/utils"

	"github.com/google/uuid"
)

type AuthService struct {
	userRepo              *repository.UserRepository
	emailVerificationRepo *repository.EmailVerificationRepository
	passwordResetRepo     *repository.PasswordResetRepository
	jwtService            *JWTService
	passwordService       *PasswordService
	emailService          *email.EmailService
}

func NewAuthService(
	userRepo *repository.UserRepository,
	emailVerificationRepo *repository.EmailVerificationRepository,
	passwordResetRepo *repository.PasswordResetRepository,
	jwtService *JWTService,
	passwordService *PasswordService,
	emailService *email.EmailService,
) *AuthService {
	return &AuthService{
		userRepo:              userRepo,
		emailVerificationRepo: emailVerificationRepo,
		passwordResetRepo:     passwordResetRepo,
		jwtService:            jwtService,
		passwordService:       passwordService,
		emailService:          emailService,
	}
}

func (s *AuthService) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	if !utils.ValidateEmail(req.Email) {
		return nil, errors.New("неверный формат email")
	}

	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("неверный email или пароль")
	}

	if user.Status != models.UserStatusActive {
		return nil, errors.New("пользователь удален")
	}

	if !s.passwordService.ComparePassword(user.PasswordHash, req.Password) {
		return nil, errors.New("неверный email или пароль")
	}

	accessToken, err := s.jwtService.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации токена: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации refresh токена: %w", err)
	}

	return &dto.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      string(user.Role),
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) Register(req dto.RegisterRequest) error {
	if !utils.ValidateEmail(req.Email) {
		return errors.New("неверный формат email")
	}

	if !utils.ValidateFullName(req.FullName) {
		return errors.New("ФИО должно содержать только русские буквы")
	}

	if err := s.passwordService.ValidatePassword(req.Password); err != nil {
		return err
	}

	if s.userRepo.EmailExists(req.Email) {
		return errors.New("пользователь с таким email уже существует")
	}

	code := generateVerificationCode()
	hashedPassword, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return err
	}

	ev := &models.EmailVerification{
		Email:        req.Email,
		Code:         code,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		CreatedAt:    time.Now(),
	}

	if err := s.emailVerificationRepo.Create(ev); err != nil {
		return fmt.Errorf("ошибка сохранения кода подтверждения: %w", err)
	}

	if err := s.emailService.SendVerificationCode(req.Email, code); err != nil {
		return fmt.Errorf("ошибка отправки email: %w", err)
	}

	return nil
}

func (s *AuthService) VerifyEmail(req dto.VerifyEmailRequest) (*dto.LoginResponse, error) {
	ev, err := s.emailVerificationRepo.FindByEmailAndCode(req.Email, req.Code)
	if err != nil {
		return nil, errors.New("неверный код подтверждения")
	}

	if ev.IsExpired() {
		return nil, errors.New("код подтверждения истек")
	}

	if s.userRepo.EmailExists(req.Email) {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	user := &models.User{
		Email:        ev.Email,
		FullName:     ev.FullName,
		PasswordHash: ev.PasswordHash,
		Role:         models.UserRoleUser,
		Status:       models.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	if err := s.emailVerificationRepo.DeleteByEmail(req.Email); err != nil {
	}

	if err := s.emailService.SendWelcomeEmail(user.Email, user.FullName); err != nil {
	}

	accessToken, err := s.jwtService.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации токена: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации refresh токена: %w", err)
	}

	return &dto.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      string(user.Role),
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) ResendVerificationCode(email string) error {
	if !utils.ValidateEmail(email) {
		return errors.New("неверный формат email")
	}

	code := generateVerificationCode()
	ev := &models.EmailVerification{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := s.emailVerificationRepo.DeleteByEmail(email); err != nil {
	}

	if err := s.emailVerificationRepo.Create(ev); err != nil {
		return fmt.Errorf("ошибка сохранения кода: %w", err)
	}

	if err := s.emailService.SendVerificationCode(email, code); err != nil {
		return fmt.Errorf("ошибка отправки email: %w", err)
	}

	return nil
}

func (s *AuthService) ForgotPassword(req dto.ForgotPasswordRequest) error {
	if !utils.ValidateEmail(req.Email) {
		return errors.New("неверный формат email")
	}

	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil
	}

	token := uuid.New().String()
	pr := &models.PasswordReset{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.passwordResetRepo.Create(pr); err != nil {
		return fmt.Errorf("ошибка создания токена сброса: %w", err)
	}

	if err := s.emailService.SendPasswordResetLink(user.Email, token); err != nil {
		return fmt.Errorf("ошибка отправки email: %w", err)
	}

	return nil
}

func (s *AuthService) ResetPassword(req dto.ResetPasswordRequest) (*dto.LoginResponse, error) {
	if err := s.passwordService.ValidatePassword(req.NewPassword); err != nil {
		return nil, err
	}

	pr, err := s.passwordResetRepo.FindByToken(req.Token)
	if err != nil {
		return nil, errors.New("неверный токен сброса пароля")
	}

	if pr.IsExpired() {
		return nil, errors.New("токен сброса пароля истек")
	}

	hashedPassword, err := s.passwordService.HashPassword(req.NewPassword)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(pr.UserID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}

	user.PasswordHash = hashedPassword
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("ошибка обновления пароля: %w", err)
	}

	if err := s.passwordResetRepo.Delete(pr.ID); err != nil {
	}

	if err := s.emailService.SendPasswordChangedNotification(user.Email); err != nil {
	}

	accessToken, err := s.jwtService.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации токена: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации refresh токена: %w", err)
	}

	return &dto.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      string(user.Role),
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) RefreshToken(req dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	claims, err := s.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("неверный refresh токен")
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}

	accessToken, err := s.jwtService.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации токена: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации refresh токена: %w", err)
	}

	return &dto.RefreshTokenResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func generateVerificationCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
