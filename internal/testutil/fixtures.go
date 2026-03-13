package testutil

import (
	"fmt"
	"testing"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/stretchr/testify/require"
)

func NewUser(t *testing.T, overrides ...func(*models.User)) *models.User {
	t.Helper()
	u := &models.User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Active:    true,
	}
	for _, fn := range overrides {
		fn(u)
	}
	return u
}

func NewUserWithPassword(t *testing.T, password string, overrides ...func(*models.User)) *models.User {
	t.Helper()
	u := NewUser(t, overrides...)
	require.NoError(t, u.SetPassword(password))
	return u
}

func UniqueEmail(prefix string) func(*models.User) {
	return func(u *models.User) {
		u.Email = fmt.Sprintf("%s@example.com", prefix)
	}
}
