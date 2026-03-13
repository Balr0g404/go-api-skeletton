package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
)

func TestSetPassword_HashesPassword(t *testing.T) {
	u := &models.User{}
	err := u.SetPassword("secret123")
	require.NoError(t, err)
	assert.NotEmpty(t, u.Password)
	assert.NotEqual(t, "secret123", u.Password)
}

func TestSetPassword_DifferentHashEachTime(t *testing.T) {
	u1, u2 := &models.User{}, &models.User{}
	require.NoError(t, u1.SetPassword("secret123"))
	require.NoError(t, u2.SetPassword("secret123"))
	assert.NotEqual(t, u1.Password, u2.Password)
}

func TestCheckPassword_Valid(t *testing.T) {
	u := &models.User{}
	require.NoError(t, u.SetPassword("correct-password"))
	assert.True(t, u.CheckPassword("correct-password"))
}

func TestCheckPassword_Invalid(t *testing.T) {
	u := &models.User{}
	require.NoError(t, u.SetPassword("correct-password"))
	assert.False(t, u.CheckPassword("wrong-password"))
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	u := &models.User{}
	require.NoError(t, u.SetPassword("secret"))
	assert.False(t, u.CheckPassword(""))
}

func TestToResponse_MapsAllFields(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	u := &models.User{
		ID:        7,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleAdmin,
		Active:    true,
		CreatedAt: now,
		Password:  "hashed-should-not-appear",
	}

	resp := u.ToResponse()

	assert.Equal(t, uint(7), resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "John", resp.FirstName)
	assert.Equal(t, "Doe", resp.LastName)
	assert.Equal(t, models.RoleAdmin, resp.Role)
	assert.True(t, resp.Active)
	assert.Equal(t, now, resp.CreatedAt)
}

func TestToResponse_InactiveUser(t *testing.T) {
	u := &models.User{Active: false, Role: models.RoleUser}
	resp := u.ToResponse()
	assert.False(t, resp.Active)
	assert.Equal(t, models.RoleUser, resp.Role)
}
