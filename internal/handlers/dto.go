package handlers

import (
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
)

type AuthResponse struct {
	User   models.UserResponse `json:"user"`
	Tokens auth.TokenPair      `json:"tokens"`
}

type TokensResponse struct {
	Tokens auth.TokenPair `json:"tokens"`
}

type UserListResponse struct {
	Users    []models.UserResponse `json:"users"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type UserCursorListResponse struct {
	Users      []models.UserResponse `json:"users"`
	NextCursor string                `json:"next_cursor,omitempty"`
	HasNext    bool                  `json:"has_next"`
	Limit      int                   `json:"limit"`
}

type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}

type SetRoleInput struct {
	Role models.Role `json:"role" binding:"required" example:"admin"`
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
