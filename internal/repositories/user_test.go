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

func TestUserRepository_ListCursor_FirstPage(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repositories.NewUserRepository(db)

	for i := range 3 {
		u := testutil.NewUserWithPassword(t, "password123", testutil.UniqueEmail(
			"cursoruser"+string(rune('a'+i)),
		))
		require.NoError(t, repo.Create(u))
	}

	users, err := repo.ListCursor(0, 10)

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
		require.NoError(t, repo.Create(u))
		if u.ID > lastID {
			lastID = u.ID
		}
	}

	// First page: limit 2
	page1, err := repo.ListCursor(0, 2)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// IDs must be ascending
	assert.Less(t, page1[0].ID, page1[1].ID)

	// Second page: after last ID of first page
	page2, err := repo.ListCursor(page1[1].ID, 2)
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
	users, err := repo.ListCursor(^uint(0)>>1, 10)

	require.NoError(t, err)
	assert.Empty(t, users)
}
