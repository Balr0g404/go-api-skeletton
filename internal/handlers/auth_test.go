package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/handlers"
	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
	"github.com/Balr0g404/go-api-skeletton/internal/mocks"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
	"github.com/Balr0g404/go-api-skeletton/pkg/pagination"
	"github.com/Balr0g404/go-api-skeletton/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type handlerSetup struct {
	handler    *handlers.AuthHandler
	repo       *mocks.UserRepository
	jwtManager *auth.JWTManager
	svc        *services.AuthService
	mr         *miniredis.Miniredis
}

func newHandlerSetup(t *testing.T) handlerSetup {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwtMgr := auth.NewJWTManager("test-secret", 1, 24)
	repo := &mocks.UserRepository{}
	mailerMock := &mocks.EmailSender{}
	mailerMock.On("Send", mock.Anything).Return(nil)
	svc := services.NewAuthService(repo, jwtMgr, client, mailerMock, "http://localhost:8080")
	h := handlers.NewAuthHandler(svc)

	return handlerSetup{handler: h, repo: repo, jwtManager: jwtMgr, svc: svc, mr: mr}
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()
	var resp response.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

func newAuthenticatedContext(c *gin.Context, userID uint, role models.Role, token string) {
	c.Set("user_id", userID)
	c.Set("user_role", string(role))
	c.Set("access_token", token)
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestRegisterHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("ExistsByEmail", "new@example.com").Return(false)
	s.repo.On("Create", mock.AnythingOfType("*models.User")).
		Run(func(args mock.Arguments) { args.Get(0).(*models.User).ID = 1 }).
		Return(nil)

	r := gin.New()
	r.POST("/register", s.handler.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{
		"email":      "new@example.com",
		"password":   "password123",
		"first_name": "John",
		"last_name":  "Doe",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)
	s.repo.AssertExpectations(t)
}

func TestRegisterHandler_InvalidJSON(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.POST("/register", s.handler.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_MissingRequiredFields(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.POST("/register", s.handler.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{
		"email": "new@example.com",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_EmailConflict(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("ExistsByEmail", "taken@example.com").Return(true)

	r := gin.New()
	r.POST("/register", s.handler.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{
		"email":      "taken@example.com",
		"password":   "password123",
		"first_name": "John",
		"last_name":  "Doe",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegisterHandler_InternalError(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("ExistsByEmail", "new@example.com").Return(false)
	s.repo.On("Create", mock.AnythingOfType("*models.User")).Return(errors.New("db error"))

	r := gin.New()
	r.POST("/register", s.handler.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{
		"email":      "new@example.com",
		"password":   "password123",
		"first_name": "John",
		"last_name":  "Doe",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestLoginHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{Email: "user@example.com", Active: true, Role: models.RoleUser}
	require.NoError(t, user.SetPassword("password123"))
	s.repo.On("FindByEmail", "user@example.com").Return(user, nil)

	r := gin.New()
	r.POST("/login", s.handler.Login)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", jsonBody(t, map[string]string{
		"email":    "user@example.com",
		"password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("FindByEmail", "nobody@example.com").Return(nil, errors.New("not found"))

	r := gin.New()
	r.POST("/login", s.handler.Login)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", jsonBody(t, map[string]string{
		"email":    "nobody@example.com",
		"password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_AccountDisabled(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{Email: "disabled@example.com", Active: false}
	require.NoError(t, user.SetPassword("password123"))
	s.repo.On("FindByEmail", "disabled@example.com").Return(user, nil)

	r := gin.New()
	r.POST("/login", s.handler.Login)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", jsonBody(t, map[string]string{
		"email":    "disabled@example.com",
		"password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ─── RefreshToken ─────────────────────────────────────────────────────────────

func TestRefreshTokenHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{ID: 1, Email: "user@example.com", Active: true, Role: models.RoleUser}
	pair, err := s.jwtManager.GenerateTokenPair(1, "user@example.com", "user")
	require.NoError(t, err)

	s.repo.On("FindByID", uint(1)).Return(user, nil)

	r := gin.New()
	r.POST("/refresh", s.handler.RefreshToken)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/refresh", jsonBody(t, map[string]string{
		"refresh_token": pair.RefreshToken,
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshTokenHandler_InvalidToken(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.POST("/refresh", s.handler.RefreshToken)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/refresh", jsonBody(t, map[string]string{
		"refresh_token": "invalid.token",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshTokenHandler_MissingBody(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.POST("/refresh", s.handler.RefreshToken)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogoutHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	pair, err := s.jwtManager.GenerateTokenPair(1, "user@example.com", "user")
	require.NoError(t, err)

	r := gin.New()
	r.POST("/logout", func(c *gin.Context) {
		newAuthenticatedContext(c, 1, models.RoleUser, pair.AccessToken)
		s.handler.Logout(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", jsonBody(t, map[string]string{
		"refresh_token": pair.RefreshToken,
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)
}

// ─── GetProfile ───────────────────────────────────────────────────────────────

func TestGetProfileHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{ID: 1, Email: "user@example.com", Role: models.RoleUser, Active: true}
	s.repo.On("FindByID", uint(1)).Return(user, nil)

	r := gin.New()
	r.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		s.handler.GetProfile(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)
}

func TestGetProfileHandler_NotFound(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("FindByID", uint(99)).Return(nil, errors.New("not found"))

	r := gin.New()
	r.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", uint(99))
		s.handler.GetProfile(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── UpdateProfile ────────────────────────────────────────────────────────────

func TestUpdateProfileHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{ID: 1, Email: "user@example.com", FirstName: "Old", Role: models.RoleUser, Active: true}
	s.repo.On("FindByID", uint(1)).Return(user, nil)
	s.repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	r := gin.New()
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		s.handler.UpdateProfile(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/profile", jsonBody(t, map[string]string{
		"first_name": "New",
		"last_name":  "Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateProfileHandler_InvalidJSON(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.PUT("/profile", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		s.handler.UpdateProfile(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ChangePassword ───────────────────────────────────────────────────────────

func TestChangePasswordHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{ID: 1, Email: "user@example.com", Active: true, Role: models.RoleUser}
	require.NoError(t, user.SetPassword("oldpassword1"))
	s.repo.On("FindByID", uint(1)).Return(user, nil)
	s.repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	r := gin.New()
	r.PUT("/password", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		s.handler.ChangePassword(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/password", jsonBody(t, map[string]string{
		"current_password": "oldpassword1",
		"new_password":     "newpassword2",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChangePasswordHandler_WrongCurrentPassword(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{ID: 1, Email: "user@example.com", Active: true, Role: models.RoleUser}
	require.NoError(t, user.SetPassword("correct1234"))
	s.repo.On("FindByID", uint(1)).Return(user, nil)

	r := gin.New()
	r.PUT("/password", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		s.handler.ChangePassword(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/password", jsonBody(t, map[string]string{
		"current_password": "wrongpassword",
		"new_password":     "newpassword2",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ListUsers ────────────────────────────────────────────────────────────────

func TestListUsersHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	users := []models.User{
		{ID: 1, Email: "a@example.com", Role: models.RoleUser, Active: true},
		{ID: 2, Email: "b@example.com", Role: models.RoleAdmin, Active: true},
	}
	s.repo.On("List", 1, 20, mock.Anything).Return(users, int64(2), nil)

	r := gin.New()
	r.GET("/admin/users", func(c *gin.Context) {
		c.Set("user_role", string(models.RoleAdmin))
		s.handler.ListUsers(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)
}

func TestListUsersHandler_DefaultPagination(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("List", 1, 20, mock.Anything).Return([]models.User{}, int64(0), nil)

	r := gin.New()
	r.GET("/admin/users", func(c *gin.Context) {
		s.handler.ListUsers(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users?page=0&page_size=200", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertCalled(t, "List", 1, 20, mock.Anything)
}

// ─── SetUserRole ──────────────────────────────────────────────────────────────

func TestSetUserRoleHandler_Success(t *testing.T) {
	s := newHandlerSetup(t)

	user := &models.User{ID: 5, Email: "user@example.com", Role: models.RoleUser, Active: true}
	s.repo.On("FindByID", uint(5)).Return(user, nil)
	s.repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	r := gin.New()
	r.PUT("/admin/users/:id/role", s.handler.SetUserRole)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/users/5/role", jsonBody(t, map[string]string{
		"role": "admin",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetUserRoleHandler_InvalidID(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.PUT("/admin/users/:id/role", s.handler.SetUserRole)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/users/abc/role", jsonBody(t, map[string]string{
		"role": "admin",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetUserRoleHandler_InvalidRole(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.PUT("/admin/users/:id/role", s.handler.SetUserRole)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/users/1/role", jsonBody(t, map[string]string{
		"role": "superuser",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetUserRoleHandler_NotFound(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("FindByID", uint(99)).Return(nil, errors.New("not found"))

	r := gin.New()
	r.PUT("/admin/users/:id/role", s.handler.SetUserRole)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/users/99/role", jsonBody(t, map[string]string{
		"role": "admin",
	}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetUserRoleHandler_MissingRole(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.PUT("/admin/users/:id/role", s.handler.SetUserRole)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/admin/users/1/role", jsonBody(t, map[string]string{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── middleware helpers used in context setup ────────────────────────────────

func TestMiddlewareHelpers_GetUserID(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", uint(42))
		assert.Equal(t, uint(42), middleware.GetUserID(c))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

func TestMiddlewareHelpers_GetUserRole(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_role", "admin")
		assert.Equal(t, "admin", middleware.GetUserRole(c))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

func TestMiddlewareHelpers_GetAccessToken(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("access_token", "my-token")
		assert.Equal(t, "my-token", middleware.GetAccessToken(c))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

// ─── ListUsersCursor ──────────────────────────────────────────────────────────

func TestListUsersCursorHandler_FirstPage(t *testing.T) {
	s := newHandlerSetup(t)

	users := []models.User{
		{ID: 1, Email: "a@example.com", Role: models.RoleUser, Active: true},
		{ID: 2, Email: "b@example.com", Role: models.RoleUser, Active: true},
	}
	// limit=2, so repo is called with limit+1=3; returns 2 → no next page
	s.repo.On("ListCursor", uint(0), 3, mock.Anything).Return(users, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor?limit=2", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, false, data["has_next"])
	assert.Nil(t, data["next_cursor"]) // omitempty: absent when empty
}

func TestListUsersCursorHandler_HasNextPage(t *testing.T) {
	s := newHandlerSetup(t)

	// limit=2, repo receives limit+1=3; returns 3 items → has_next=true
	users := []models.User{
		{ID: 1, Email: "a@example.com", Role: models.RoleUser, Active: true},
		{ID: 2, Email: "b@example.com", Role: models.RoleUser, Active: true},
		{ID: 3, Email: "c@example.com", Role: models.RoleUser, Active: true},
	}
	s.repo.On("ListCursor", uint(0), 3, mock.Anything).Return(users, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor?limit=2", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, true, data["has_next"])
	assert.NotEmpty(t, data["next_cursor"])
}

func TestListUsersCursorHandler_WithCursor(t *testing.T) {
	s := newHandlerSetup(t)

	cursor := pagination.EncodeCursor(5)
	users := []models.User{
		{ID: 6, Email: "f@example.com", Role: models.RoleUser, Active: true},
	}
	s.repo.On("ListCursor", uint(5), 21, mock.Anything).Return(users, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/admin/users/cursor?cursor=%s", cursor), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResponse(t, w)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, false, data["has_next"])
}

func TestListUsersCursorHandler_InvalidCursor(t *testing.T) {
	s := newHandlerSetup(t)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor?cursor=!!!invalid!!!", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListUsersCursorHandler_DefaultLimit(t *testing.T) {
	s := newHandlerSetup(t)

	// No limit param → default 20 → repo called with 21
	s.repo.On("ListCursor", uint(0), 21, mock.Anything).Return([]models.User{}, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertCalled(t, "ListCursor", uint(0), 21, mock.Anything)
}

func TestListUsersCursorHandler_LimitClamped(t *testing.T) {
	s := newHandlerSetup(t)

	// limit=0 → clamped to default 20
	s.repo.On("ListCursor", uint(0), 21, mock.Anything).Return([]models.User{}, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor?limit=0", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertCalled(t, "ListCursor", uint(0), 21, mock.Anything)
}

func TestListUsersCursorHandler_NextCursorIsDecodable(t *testing.T) {
	s := newHandlerSetup(t)

	users := []models.User{
		{ID: 10, Email: "x@example.com", Role: models.RoleUser, Active: true},
		{ID: 11, Email: "y@example.com", Role: models.RoleUser, Active: true},
	}
	// limit=1, repo receives 2; returns 2 → has_next=true, next_cursor encodes ID 10
	s.repo.On("ListCursor", uint(0), 2, mock.Anything).Return(users, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor?limit=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data := decodeResponse(t, w).Data.(map[string]interface{})

	nextCursor := data["next_cursor"].(string)
	decoded, err := pagination.DecodeCursor(nextCursor)
	assert.NoError(t, err)
	assert.Equal(t, uint(10), decoded)
}

// ─── Filtering / sorting on ListUsers ─────────────────────────────────────────

func TestListUsersHandler_FilterByRole(t *testing.T) {
	s := newHandlerSetup(t)

	users := []models.User{
		{ID: 1, Email: "admin@example.com", Role: models.RoleAdmin, Active: true},
	}
	s.repo.On("List", 1, 20, filtering.Options{
		Sort: "id", Order: filtering.OrderAsc,
		Filters: map[string]string{"role": "admin"},
	}).Return(users, int64(1), nil)

	r := gin.New()
	r.GET("/admin/users", s.handler.ListUsers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users?filter[role]=admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertExpectations(t)
}

func TestListUsersHandler_SortByEmail(t *testing.T) {
	s := newHandlerSetup(t)

	s.repo.On("List", 1, 20, filtering.Options{
		Sort: "email", Order: filtering.OrderDesc,
		Filters: map[string]string{},
	}).Return([]models.User{}, int64(0), nil)

	r := gin.New()
	r.GET("/admin/users", s.handler.ListUsers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users?sort=email&order=desc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertExpectations(t)
}

func TestListUsersHandler_InvalidSortFallsBackToDefault(t *testing.T) {
	s := newHandlerSetup(t)

	// unknown sort → falls back to "id"
	s.repo.On("List", 1, 20, filtering.Options{
		Sort: "id", Order: filtering.OrderAsc,
		Filters: map[string]string{},
	}).Return([]models.User{}, int64(0), nil)

	r := gin.New()
	r.GET("/admin/users", s.handler.ListUsers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users?sort=injected_col", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertExpectations(t)
}

func TestListUsersHandler_UnknownFilterIgnored(t *testing.T) {
	s := newHandlerSetup(t)

	// unknown filter key → silently dropped, no filter applied
	s.repo.On("List", 1, 20, filtering.Options{
		Sort: "id", Order: filtering.OrderAsc,
		Filters: map[string]string{},
	}).Return([]models.User{}, int64(0), nil)

	r := gin.New()
	r.GET("/admin/users", s.handler.ListUsers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users?filter[unknown_col]=value", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertExpectations(t)
}

// ─── ForgotPassword / ResetPassword handlers ──────────────────────────────────

func TestForgotPasswordHandler(t *testing.T) {
	s := newHandlerSetup(t)
	u := &models.User{Email: "forgot@example.com", FirstName: "Foo", LastName: "Bar"}
	u.ID = 7
	require.NoError(t, u.SetPassword("password123"))
	s.repo.On("FindByEmail", "forgot@example.com").Return(u, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", jsonBody(t, map[string]string{"email": "forgot@example.com"}))
	r.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	s.handler.ForgotPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var body response.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.True(t, body.Success)
}

func TestForgotPasswordHandler_InvalidBody(t *testing.T) {
	s := newHandlerSetup(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", jsonBody(t, map[string]string{"email": "not-an-email"}))
	r.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	s.handler.ForgotPassword(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_InvalidToken(t *testing.T) {
	s := newHandlerSetup(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/reset-password", jsonBody(t, map[string]string{
		"token": "badtoken", "password": "newpassword123",
	}))
	r.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	s.handler.ResetPassword(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListUsersCursorHandler_FilterByActive(t *testing.T) {
	s := newHandlerSetup(t)

	users := []models.User{
		{ID: 1, Email: "active@example.com", Role: models.RoleUser, Active: true},
	}
	s.repo.On("ListCursor", uint(0), 21, filtering.Options{
		Sort: "id", Order: filtering.OrderAsc,
		Filters: map[string]string{"active": "true"},
	}).Return(users, nil)

	r := gin.New()
	r.GET("/admin/users/cursor", s.handler.ListUsersCursor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/users/cursor?filter[active]=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	s.repo.AssertExpectations(t)
}
