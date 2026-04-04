package mocks

import (
	"context"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
	"github.com/stretchr/testify/mock"
)

type UserRepository struct {
	mock.Mock
}

func (m *UserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *UserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *UserRepository) List(ctx context.Context, page, pageSize int, opts filtering.Options) ([]models.User, int64, error) {
	args := m.Called(ctx, page, pageSize, opts)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *UserRepository) ListCursor(ctx context.Context, afterID uint, limit int, opts filtering.Options) ([]models.User, error) {
	args := m.Called(ctx, afterID, limit, opts)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *UserRepository) ExistsByEmail(ctx context.Context, email string) bool {
	args := m.Called(ctx, email)
	return args.Bool(0)
}
