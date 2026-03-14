package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/mocks"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/internal/testutil"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
)

func newTestService(t *testing.T, repo *mocks.UserRepository) (*services.AuthService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwt := auth.NewJWTManager("test-secret", 1, 24)
	mailer := &mocks.EmailSender{}
	mailer.On("Send", mock.Anything).Return(nil)
	return services.NewAuthService(repo, jwt, client, mailer, "http://localhost:8080"), mr
}

func TestRegister_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("ExistsByEmail", "test@example.com").Return(false)
	repo.On("Create", mock.AnythingOfType("*models.User")).
		Run(func(args mock.Arguments) {
			args.Get(0).(*models.User).ID = 1
		}).
		Return(nil)

	user, tokens, err := svc.Register(services.RegisterInput{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	})

	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	repo.AssertExpectations(t)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("ExistsByEmail", "test@example.com").Return(true)

	_, _, err := svc.Register(services.RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, services.ErrEmailAlreadyExists)
	repo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUserWithPassword(t, "password123")
	u.ID = 1

	repo.On("FindByEmail", u.Email).Return(u, nil)

	user, tokens, err := svc.Login(services.LoginInput{
		Email:    u.Email,
		Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, u.Email, user.Email)
	assert.NotEmpty(t, tokens.AccessToken)
	repo.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("FindByEmail", "unknown@example.com").Return(nil, errors.New("not found"))

	_, _, err := svc.Login(services.LoginInput{
		Email:    "unknown@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	repo.AssertExpectations(t)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUserWithPassword(t, "correct-password")
	u.ID = 1

	repo.On("FindByEmail", u.Email).Return(u, nil)

	_, _, err := svc.Login(services.LoginInput{
		Email:    u.Email,
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	repo.AssertExpectations(t)
}

func TestLogin_AccountDisabled(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUserWithPassword(t, "password123", func(u *models.User) {
		u.Active = false
	})
	u.ID = 1

	repo.On("FindByEmail", u.Email).Return(u, nil)

	_, _, err := svc.Login(services.LoginInput{
		Email:    u.Email,
		Password: "password123",
	})

	assert.ErrorIs(t, err, services.ErrAccountDisabled)
	repo.AssertExpectations(t)
}

func TestRefreshTokens_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUser(t)
	u.ID = 1

	jwtManager := auth.NewJWTManager("test-secret", 1, 24)
	tokens, err := jwtManager.GenerateTokenPair(u.ID, u.Email, string(u.Role))
	require.NoError(t, err)

	repo.On("FindByID", u.ID).Return(u, nil)

	newTokens, err := svc.RefreshTokens(services.RefreshInput{RefreshToken: tokens.RefreshToken})

	require.NoError(t, err)
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	repo.AssertExpectations(t)
}

func TestRefreshTokens_BlacklistedToken(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	jwtManager := auth.NewJWTManager("test-secret", 1, 24)
	tokens, err := jwtManager.GenerateTokenPair(1, "test@example.com", "user")
	require.NoError(t, err)

	mr.Set("blacklist:"+tokens.RefreshToken, "1")

	_, err = svc.RefreshTokens(services.RefreshInput{RefreshToken: tokens.RefreshToken})

	assert.ErrorIs(t, err, services.ErrTokenBlacklisted)
}

func TestLogout_BlacklistsBothTokens(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	jwtManager := auth.NewJWTManager("test-secret", 1, 24)
	tokens, err := jwtManager.GenerateTokenPair(1, "test@example.com", "user")
	require.NoError(t, err)

	svc.Logout(tokens.AccessToken, tokens.RefreshToken)

	assert.True(t, mr.Exists("blacklist:"+tokens.AccessToken))
	assert.True(t, mr.Exists("blacklist:"+tokens.RefreshToken))
}

func TestGetProfile_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUser(t)
	u.ID = 1

	repo.On("FindByID", uint(1)).Return(u, nil)

	profile, err := svc.GetProfile(1)

	require.NoError(t, err)
	assert.Equal(t, u.Email, profile.Email)
	repo.AssertExpectations(t)
}

func TestGetProfile_NotFound(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("FindByID", uint(99)).Return(nil, errors.New("not found"))

	_, err := svc.GetProfile(99)

	assert.ErrorIs(t, err, services.ErrUserNotFound)
	repo.AssertExpectations(t)
}

func TestUpdateProfile_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUser(t)
	u.ID = 1

	repo.On("FindByID", uint(1)).Return(u, nil)
	repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	profile, err := svc.UpdateProfile(1, services.UpdateProfileInput{
		FirstName: "Updated",
		LastName:  "Name",
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", profile.FirstName)
	assert.Equal(t, "Name", profile.LastName)
	repo.AssertExpectations(t)
}

func TestChangePassword_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUserWithPassword(t, "old-password")
	u.ID = 1

	repo.On("FindByID", uint(1)).Return(u, nil)
	repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	err := svc.ChangePassword(1, services.ChangePasswordInput{
		CurrentPassword: "old-password",
		NewPassword:     "new-password",
	})

	require.NoError(t, err)
	assert.True(t, u.CheckPassword("new-password"))
	repo.AssertExpectations(t)
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUserWithPassword(t, "correct-password")
	u.ID = 1

	repo.On("FindByID", uint(1)).Return(u, nil)

	err := svc.ChangePassword(1, services.ChangePasswordInput{
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password",
	})

	assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	repo.AssertExpectations(t)
}

func TestListUsers_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	users := []models.User{
		*testutil.NewUser(t, testutil.UniqueEmail("user1")),
		*testutil.NewUser(t, testutil.UniqueEmail("user2")),
	}

	repo.On("List", 1, 20, mock.Anything).Return(users, int64(2), nil)

	result, total, err := svc.ListUsers(1, 20, filtering.Options{})

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestSetUserRole_Success(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	u := testutil.NewUser(t)
	u.ID = 1

	repo.On("FindByID", uint(1)).Return(u, nil)
	repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	result, err := svc.SetUserRole(1, models.RoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, models.RoleAdmin, result.Role)
	repo.AssertExpectations(t)
}

func TestSetUserRole_NotFound(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("FindByID", uint(99)).Return(nil, errors.New("not found"))

	_, err := svc.SetUserRole(99, models.RoleAdmin)

	assert.ErrorIs(t, err, services.ErrUserNotFound)
	repo.AssertExpectations(t)
}

// ─── ListUsersCursor ──────────────────────────────────────────────────────────

func TestListUsersCursor_FirstPage(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	users := []models.User{
		{ID: 1, Email: "a@example.com", Role: models.RoleUser, Active: true},
		{ID: 2, Email: "b@example.com", Role: models.RoleUser, Active: true},
	}
	// limit=2, service fetches limit+1=3; returns 2 → no next page
	repo.On("ListCursor", uint(0), 3, mock.Anything).Return(users, nil)

	result, nextCursor, hasNext, err := svc.ListUsersCursor("", 2, filtering.Options{})

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.False(t, hasNext)
	assert.Empty(t, nextCursor)
	repo.AssertExpectations(t)
}

func TestListUsersCursor_HasNextPage(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	// limit=2, returns 3 → has_next=true, only first 2 returned
	users := []models.User{
		{ID: 1, Email: "a@example.com", Role: models.RoleUser, Active: true},
		{ID: 2, Email: "b@example.com", Role: models.RoleUser, Active: true},
		{ID: 3, Email: "c@example.com", Role: models.RoleUser, Active: true},
	}
	repo.On("ListCursor", uint(0), 3, mock.Anything).Return(users, nil)

	result, nextCursor, hasNext, err := svc.ListUsersCursor("", 2, filtering.Options{})

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.True(t, hasNext)
	assert.NotEmpty(t, nextCursor)
	// next cursor should encode ID 2 (last item returned)
	assert.Equal(t, uint(2), result[len(result)-1].ID)
}

func TestListUsersCursor_InvalidCursor(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	_, _, _, err := svc.ListUsersCursor("!!!invalid!!!", 20, filtering.Options{})

	assert.Error(t, err)
	repo.AssertNotCalled(t, "ListCursor")
}

func TestListUsersCursor_EmptyResult(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("ListCursor", uint(0), 21, mock.Anything).Return([]models.User{}, nil)

	result, nextCursor, hasNext, err := svc.ListUsersCursor("", 20, filtering.Options{})

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.False(t, hasNext)
	assert.Empty(t, nextCursor)
}

// ─── ListUsers with filtering ────────────────────────────────────────────────

func TestListUsers_WithFilter(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	opts := filtering.Options{
		Sort:    "email",
		Order:   filtering.OrderAsc,
		Filters: map[string]string{"role": "admin"},
	}

	users := []models.User{
		{ID: 1, Email: "admin@example.com", Role: models.RoleAdmin, Active: true},
	}
	repo.On("List", 1, 20, opts).Return(users, int64(1), nil)

	result, total, err := svc.ListUsers(1, 20, opts)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
	assert.Equal(t, models.RoleAdmin, result[0].Role)
	repo.AssertExpectations(t)
}

func TestListUsersCursor_WithFilter(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	opts := filtering.Options{
		Sort:    "id",
		Order:   filtering.OrderAsc,
		Filters: map[string]string{"active": "true"},
	}

	users := []models.User{
		{ID: 3, Email: "active@example.com", Role: models.RoleUser, Active: true},
	}
	repo.On("ListCursor", uint(0), 21, opts).Return(users, nil)

	result, _, hasNext, err := svc.ListUsersCursor("", 20, opts)

	require.NoError(t, err)
	assert.False(t, hasNext)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

// ─── ForgotPassword / ResetPassword ──────────────────────────────────────────

func TestForgotPassword_UserExists(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	u := testutil.NewUser(t, testutil.UniqueEmail("reset"))
	u.ID = 5
	repo.On("FindByEmail", u.Email).Return(u, nil)

	err := svc.ForgotPassword(services.ForgotPasswordInput{Email: u.Email})
	assert.NoError(t, err)
	// token should be stored in Redis
	keys := mr.Keys()
	found := false
	for _, k := range keys {
		if len(k) > 9 && k[:9] == "pwd_reset" {
			found = true
		}
	}
	assert.True(t, found, "expected pwd_reset key in Redis")
	repo.AssertExpectations(t)
}

func TestForgotPassword_UserNotFound(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	repo.On("FindByEmail", "ghost@example.com").Return(nil, errors.New("not found"))

	err := svc.ForgotPassword(services.ForgotPasswordInput{Email: "ghost@example.com"})
	assert.NoError(t, err) // always nil for security
	repo.AssertExpectations(t)
}

func TestResetPassword_ValidToken(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	// Seed a reset token in miniredis
	mr.Set("pwd_reset:validtoken123", "42")
	mr.SetTTL("pwd_reset:validtoken123", 3600*time.Second)

	u := testutil.NewUserWithPassword(t, "oldpass", testutil.UniqueEmail("someone"))
	u.ID = 42
	repo.On("FindByID", uint(42)).Return(u, nil)
	repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	err := svc.ResetPassword(services.ResetPasswordInput{Token: "validtoken123", Password: "newpassword123"})
	assert.NoError(t, err)

	// token should be deleted
	assert.False(t, mr.Exists("pwd_reset:validtoken123"), "key should be gone")
	repo.AssertExpectations(t)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	err := svc.ResetPassword(services.ResetPasswordInput{Token: "badtoken", Password: "newpass123"})
	assert.ErrorIs(t, err, services.ErrInvalidResetToken)
}

func TestResetPassword_CorruptTokenData(t *testing.T) {
	// Redis has the key but the value is not a valid uint (corrupt data).
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	mr.Set("pwd_reset:corrupttoken", "not-a-number")

	err := svc.ResetPassword(services.ResetPasswordInput{Token: "corrupttoken", Password: "newpass123"})
	assert.ErrorIs(t, err, services.ErrInvalidResetToken)
}

func TestResetPassword_UserNotFoundAfterToken(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	mr.Set("pwd_reset:mytoken", "999")
	repo.On("FindByID", uint(999)).Return(nil, errors.New("not found"))

	err := svc.ResetPassword(services.ResetPasswordInput{Token: "mytoken", Password: "newpass123"})
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestResetPassword_UpdateFails(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	mr.Set("pwd_reset:mytoken", "42")
	u := testutil.NewUserWithPassword(t, "oldpass", testutil.UniqueEmail("updfail"))
	u.ID = 42
	repo.On("FindByID", uint(42)).Return(u, nil)
	repo.On("Update", mock.AnythingOfType("*models.User")).Return(errors.New("db error"))

	err := svc.ResetPassword(services.ResetPasswordInput{Token: "mytoken", Password: "newpassword123"})
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// ─── IsTokenBlacklisted ───────────────────────────────────────────────────

func TestIsTokenBlacklisted_False(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	assert.False(t, svc.IsTokenBlacklisted("some-token"))
}

func TestIsTokenBlacklisted_True(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, mr := newTestService(t, repo)

	mr.Set("blacklist:some-token", "1")

	assert.True(t, svc.IsTokenBlacklisted("some-token"))
}

// ─── RefreshTokens — missing branch ──────────────────────────────────────

func TestRefreshTokens_AccountDisabled(t *testing.T) {
	repo := &mocks.UserRepository{}
	svc, _ := newTestService(t, repo)

	jwtManager := auth.NewJWTManager("test-secret", 1, 24)
	tokens, err := jwtManager.GenerateTokenPair(1, "test@example.com", "user")
	require.NoError(t, err)

	u := testutil.NewUser(t)
	u.ID = 1
	u.Active = false
	repo.On("FindByID", uint(1)).Return(u, nil)

	_, err = svc.RefreshTokens(services.RefreshInput{RefreshToken: tokens.RefreshToken})
	assert.ErrorIs(t, err, services.ErrAccountDisabled)
	repo.AssertExpectations(t)
}
