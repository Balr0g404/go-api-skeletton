package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenBlacklisted   = errors.New("token has been revoked")
)

type AuthService struct {
	userRepo   UserRepository
	jwtManager *auth.JWTManager
	redis      *redis.Client
}

func NewAuthService(userRepo UserRepository, jwtManager *auth.JWTManager, redis *redis.Client) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		redis:      redis,
	}
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

func (s *AuthService) ListUsers(page, pageSize int) ([]models.UserResponse, int64, error) {
	users, total, err := s.userRepo.List(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}
	return responses, total, nil
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

func (s *AuthService) CacheUser(user *models.UserResponse, ttl time.Duration) {
	data, err := json.Marshal(user)
	if err != nil {
		return
	}
	key := fmt.Sprintf("user:%d", user.ID)
	s.redis.Set(context.Background(), key, data, ttl)
}

func (s *AuthService) GetCachedUser(userID uint) *models.UserResponse {
	key := fmt.Sprintf("user:%d", userID)
	data, err := s.redis.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil
	}
	var user models.UserResponse
	if err := json.Unmarshal(data, &user); err != nil {
		return nil
	}
	return &user
}