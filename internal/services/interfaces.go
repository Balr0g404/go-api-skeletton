package services

import (
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	List(page, pageSize int, opts filtering.Options) ([]models.User, int64, error)
	ListCursor(afterID uint, limit int, opts filtering.Options) ([]models.User, error)
	ExistsByEmail(email string) bool
}
