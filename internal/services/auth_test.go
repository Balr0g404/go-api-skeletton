package services_test

import (
	"errors"
	"testing"

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
)

func newTestService(t *testing.T, repo *mocks.UserRepository) (*services.AuthService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwt := auth.NewJWTManager("test-secret", 1, 24)
	return services.NewAuthService(repo, jwt, client), mr
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

	repo.On("List", 1, 20).Return(users, int64(2), nil)

	result, total, err := svc.ListUsers(1, 20)

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
