package services

import "github.com/Balr0g404/go-api-skeletton/internal/models"

type UserRepository interface {
	Create(user *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	List(page, pageSize int) ([]models.User, int64, error)
	ExistsByEmail(email string) bool
}
