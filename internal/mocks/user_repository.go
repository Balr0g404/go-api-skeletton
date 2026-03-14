package mocks

import (
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
	"github.com/stretchr/testify/mock"
)

type UserRepository struct {
	mock.Mock
}

func (m *UserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *UserRepository) FindByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *UserRepository) FindByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *UserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *UserRepository) List(page, pageSize int, opts filtering.Options) ([]models.User, int64, error) {
	args := m.Called(page, pageSize, opts)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *UserRepository) ListCursor(afterID uint, limit int, opts filtering.Options) ([]models.User, error) {
	args := m.Called(afterID, limit, opts)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *UserRepository) ExistsByEmail(email string) bool {
	args := m.Called(email)
	return args.Bool(0)
}
