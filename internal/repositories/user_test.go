//go:build integration

package repositories_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/repositories"
	"github.com/Balr0g404/go-api-skeletton/internal/testutil"
)

func TestUserRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")

	err := repo.Create(u)

	require.NoError(t, err)
	assert.NotZero(t, u.ID)
	assert.NotZero(t, u.CreatedAt)
}

func TestUserRepository_FindByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(u))

	found, err := repo.FindByID(u.ID)

	require.NoError(t, err)
	assert.Equal(t, u.ID, found.ID)
	assert.Equal(t, u.Email, found.Email)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	_, err := repo.FindByID(99999)

	assert.Error(t, err)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(u))

	found, err := repo.FindByEmail(u.Email)

	require.NoError(t, err)
	assert.Equal(t, u.ID, found.ID)
}

func TestUserRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(u))

	u.FirstName = "Updated"
	require.NoError(t, repo.Update(u))

	found, err := repo.FindByID(u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", found.FirstName)
}

func TestUserRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(u))

	require.NoError(t, repo.Delete(u.ID))

	_, err := repo.FindByID(u.ID)
	assert.Error(t, err)
}

func TestUserRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	for i := range 3 {
		u := testutil.NewUserWithPassword(t, "password123", testutil.UniqueEmail(
			"listuser"+string(rune('a'+i)),
		))
		require.NoError(t, repo.Create(u))
	}

	users, total, err := repo.List(1, 10)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(3))
	assert.NotEmpty(t, users)
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(u))

	assert.True(t, repo.ExistsByEmail(u.Email))
	assert.False(t, repo.ExistsByEmail("nobody@example.com"))
}
