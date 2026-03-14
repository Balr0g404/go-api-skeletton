package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/email"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
	"github.com/Balr0g404/go-api-skeletton/pkg/pagination"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenBlacklisted   = errors.New("token has been revoked")
	ErrInvalidResetToken  = errors.New("invalid or expired reset token")
)

type AuthService struct {
	userRepo   UserRepository
	jwtManager *auth.JWTManager
	redis      *redis.Client
	mailer     email.Sender
	baseURL    string
}

func NewAuthService(userRepo UserRepository, jwtManager *auth.JWTManager, redis *redis.Client, mailer email.Sender, baseURL string) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		redis:      redis,
		mailer:     mailer,
		baseURL:    baseURL,
	}
}

type ForgotPasswordInput struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordInput struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type RegisterInput struct {
	Email     string `json:"email" binding:"required,email" example:"user@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"password123"`
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" binding:"required" example:"Doe"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIs..."`
}

type UpdateProfileInput struct {
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldpassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=8" example:"newpassword456"`
}

func (s *AuthService) Register(input RegisterInput) (*models.UserResponse, *auth.TokenPair, error) {
	if s.userRepo.ExistsByEmail(input.Email) {
		return nil, nil, ErrEmailAlreadyExists
	}

	user := &models.User{
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Role:      models.RoleUser,
		Active:    true,
	}

	if err := user.SetPassword(input.Password); err != nil {
		return nil, nil, err
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, nil, err
	}

	go func() {
		if err := s.mailer.Send(email.Welcome(user.FirstName, user.Email)); err != nil {
			log.Warn().Err(err).Str("to", user.Email).Msg("failed to send welcome email")
		}
	}()

	tokens, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, nil, err
	}

	resp := user.ToResponse()
	return &resp, tokens, nil
}

func (s *AuthService) Login(input LoginInput) (*models.UserResponse, *auth.TokenPair, error) {
	user, err := s.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if !user.Active {
		return nil, nil, ErrAccountDisabled
	}

	if !user.CheckPassword(input.Password) {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, nil, err
	}

	resp := user.ToResponse()
	return &resp, tokens, nil
}

func (s *AuthService) RefreshTokens(input RefreshInput) (*auth.TokenPair, error) {
	if s.isTokenBlacklisted(input.RefreshToken) {
		return nil, ErrTokenBlacklisted
	}

	claims, err := s.jwtManager.ValidateToken(input.RefreshToken, auth.RefreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.Active {
		return nil, ErrAccountDisabled
	}

	s.blacklistToken(input.RefreshToken, time.Until(claims.ExpiresAt.Time))

	return s.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Role))
}

func (s *AuthService) Logout(accessToken, refreshToken string) {
	if claims, err := s.jwtManager.ValidateToken(accessToken, auth.AccessToken); err == nil {
		s.blacklistToken(accessToken, time.Until(claims.ExpiresAt.Time))
	}
	if refreshToken != "" {
		if claims, err := s.jwtManager.ValidateToken(refreshToken, auth.RefreshToken); err == nil {
			s.blacklistToken(refreshToken, time.Until(claims.ExpiresAt.Time))
		}
	}
}

func (s *AuthService) GetProfile(userID uint) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	resp := user.ToResponse()
	return &resp, nil
}

func (s *AuthService) UpdateProfile(userID uint, input UpdateProfileInput) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if input.FirstName != "" {
		user.FirstName = input.FirstName
	}
	if input.LastName != "" {
		user.LastName = input.LastName
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	resp := user.ToResponse()
	return &resp, nil
}

func (s *AuthService) ChangePassword(userID uint, input ChangePasswordInput) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.CheckPassword(input.CurrentPassword) {
		return ErrInvalidCredentials
	}

	if err := user.SetPassword(input.NewPassword); err != nil {
		return err
	}

	return s.userRepo.Update(user)
}

func (s *AuthService) ListUsers(page, pageSize int, opts filtering.Options) ([]models.UserResponse, int64, error) {
	users, total, err := s.userRepo.List(page, pageSize, opts)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}
	return responses, total, nil
}

// ListUsersCursor returns a page of users using cursor-based pagination.
// cursorStr is an opaque base64-encoded ID; pass "" for the first page.
// Returns: users, nextCursor (empty if no more pages), hasNext, error.
func (s *AuthService) ListUsersCursor(cursorStr string, limit int, opts filtering.Options) ([]models.UserResponse, string, bool, error) {
	afterID, err := pagination.DecodeCursor(cursorStr)
	if err != nil {
		return nil, "", false, err
	}

	// Fetch one extra to detect whether a next page exists.
	users, err := s.userRepo.ListCursor(afterID, limit+1, opts)
	if err != nil {
		return nil, "", false, err
	}

	hasNext := len(users) > limit
	if hasNext {
		users = users[:limit]
	}

	responses := make([]models.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	var nextCursor string
	if hasNext {
		nextCursor = pagination.EncodeCursor(users[len(users)-1].ID)
	}

	return responses, nextCursor, hasNext, nil
}

func (s *AuthService) SetUserRole(userID uint, role models.Role) (*models.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	user.Role = role
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	resp := user.ToResponse()
	return &resp, nil
}

func (s *AuthService) IsTokenBlacklisted(token string) bool {
	return s.isTokenBlacklisted(token)
}

func (s *AuthService) blacklistToken(token string, expiration time.Duration) {
	key := fmt.Sprintf("blacklist:%s", token)
	s.redis.Set(context.Background(), key, true, expiration)
}

func (s *AuthService) isTokenBlacklisted(token string) bool {
	key := fmt.Sprintf("blacklist:%s", token)
	result, err := s.redis.Exists(context.Background(), key).Result()
	return err == nil && result > 0
}

func (s *AuthService) ForgotPassword(input ForgotPasswordInput) error {
	user, err := s.userRepo.FindByEmail(input.Email)
	if err != nil {
		// Security: don't reveal whether the email is registered
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Error().Err(err).Msg("failed to generate password reset token")
		return nil
	}
	token := hex.EncodeToString(tokenBytes)

	key := fmt.Sprintf("pwd_reset:%s", token)
	s.redis.Set(context.Background(), key, strconv.FormatUint(uint64(user.ID), 10), time.Hour)

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	go func() {
		if err := s.mailer.Send(email.PasswordReset(user.FirstName, resetURL)); err != nil {
			log.Warn().Err(err).Str("to", user.Email).Msg("failed to send password reset email")
		}
	}()

	return nil
}

func (s *AuthService) ResetPassword(input ResetPasswordInput) error {
	key := fmt.Sprintf("pwd_reset:%s", input.Token)
	val, err := s.redis.Get(context.Background(), key).Result()
	if err != nil {
		return ErrInvalidResetToken
	}

	userID64, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return ErrInvalidResetToken
	}

	user, err := s.userRepo.FindByID(uint(userID64))
	if err != nil {
		return ErrInvalidResetToken
	}

	if err := user.SetPassword(input.Password); err != nil {
		return err
	}

	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	s.redis.Del(context.Background(), key)
	return nil
}
