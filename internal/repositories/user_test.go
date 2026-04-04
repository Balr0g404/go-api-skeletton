//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Balr0g404/go-api-skeletton/internal/repositories"
	"github.com/Balr0g404/go-api-skeletton/internal/testutil"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
)

func TestUserRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")

	err := repo.Create(context.Background(), u)

	require.NoError(t, err)
	assert.NotZero(t, u.ID)
	assert.NotZero(t, u.CreatedAt)
}

// TestUserRepository_Create_UniqueConstraint vérifie que la contrainte unique
// sur email retourne gorm.ErrDuplicatedKey (pas de panic, erreur structurée).
func TestUserRepository_Create_UniqueConstraint(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(context.Background(), u))

	// Second insert with same email should fail with ErrDuplicatedKey
	u2 := testutil.NewUserWithPassword(t, "password456")
	u2.Email = u.Email // same email
	err := repo.Create(context.Background(), u2)

	require.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestUserRepository_FindByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(context.Background(), u))

	found, err := repo.FindByID(context.Background(), u.ID)

	require.NoError(t, err)
	assert.Equal(t, u.ID, found.ID)
	assert.Equal(t, u.Email, found.Email)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	_, err := repo.FindByID(context.Background(), 99999)

	assert.Error(t, err)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(context.Background(), u))

	found, err := repo.FindByEmail(context.Background(), u.Email)

	require.NoError(t, err)
	assert.Equal(t, u.ID, found.ID)
}

func TestUserRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(context.Background(), u))

	u.FirstName = "Updated"
	require.NoError(t, repo.Update(context.Background(), u))

	found, err := repo.FindByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", found.FirstName)
}

func TestUserRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(context.Background(), u))

	require.NoError(t, repo.Delete(context.Background(), u.ID))

	_, err := repo.FindByID(context.Background(), u.ID)
	assert.Error(t, err)
}

func TestUserRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	for i := range 3 {
		u := testutil.NewUserWithPassword(t, "password123", testutil.UniqueEmail(
			"listuser"+string(rune('a'+i)),
		))
		require.NoError(t, repo.Create(context.Background(), u))
	}

	users, total, err := repo.List(context.Background(), 1, 10, filtering.Options{})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(3))
	assert.NotEmpty(t, users)
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	u := testutil.NewUserWithPassword(t, "password123")
	require.NoError(t, repo.Create(context.Background(), u))

	assert.True(t, repo.ExistsByEmail(context.Background(), u.Email))
	assert.False(t, repo.ExistsByEmail(context.Background(), "nobody@example.com"))
}

func TestUserRepository_ListCursor_FirstPage(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	for i := range 3 {
		u := testutil.NewUserWithPassword(t, "password123", testutil.UniqueEmail(
			"cursoruser"+string(rune('a'+i)),
		))
		require.NoError(t, repo.Create(context.Background(), u))
	}

	users, err := repo.ListCursor(context.Background(), 0, 10, filtering.Options{})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 3)
}

func TestUserRepository_ListCursor_Pagination(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	// Create 4 users to paginate through
	var lastID uint
	for i := range 4 {
		u := testutil.NewUserWithPassword(t, "password123", testutil.UniqueEmail(
			"pguser"+string(rune('a'+i)),
		))
		require.NoError(t, repo.Create(context.Background(), u))
		if u.ID > lastID {
			lastID = u.ID
		}
	}

	// First page: limit 2
	page1, err := repo.ListCursor(context.Background(), 0, 2, filtering.Options{})
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// IDs must be ascending
	assert.Less(t, page1[0].ID, page1[1].ID)

	// Second page: after last ID of first page
	page2, err := repo.ListCursor(context.Background(), page1[1].ID, 2, filtering.Options{})
	require.NoError(t, err)
	// Every ID in page2 must be greater than the cursor
	for _, u := range page2 {
		assert.Greater(t, u.ID, page1[1].ID)
	}
}

func TestUserRepository_ListCursor_EmptyAfterLastID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	// Use a very large afterID to ensure no results
	users, err := repo.ListCursor(context.Background(), ^uint(0)>>1, 10, filtering.Options{})

	require.NoError(t, err)
	assert.Empty(t, users)
}
