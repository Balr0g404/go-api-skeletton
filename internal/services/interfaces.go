package services

import (
	"context"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	List(ctx context.Context, page, pageSize int, opts filtering.Options) ([]models.User, int64, error)
	ListCursor(ctx context.Context, afterID uint, limit int, opts filtering.Options) ([]models.User, error)
	ExistsByEmail(ctx context.Context, email string) bool
}
