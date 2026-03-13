package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func callHandler(fn func(*gin.Context)) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	fn(c)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()
	var resp response.APIResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

func TestOK(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.OK(c, gin.H{"key": "value"})
	})
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decode(t, w)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestCreated(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.Created(c, gin.H{"id": 1})
	})
	assert.Equal(t, http.StatusCreated, w.Code)
	resp := decode(t, w)
	assert.True(t, resp.Success)
}

func TestMessage(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.Message(c, "operation successful")
	})
	assert.Equal(t, http.StatusOK, w.Code)
	resp := decode(t, w)
	assert.True(t, resp.Success)
	assert.Equal(t, "operation successful", resp.Message)
	assert.Nil(t, resp.Data)
}

func TestBadRequest(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.BadRequest(c, "invalid input")
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decode(t, w)
	assert.False(t, resp.Success)
	assert.Equal(t, "invalid input", resp.Error)
}

func TestUnauthorized(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.Unauthorized(c, "token required")
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := decode(t, w)
	assert.False(t, resp.Success)
	assert.Equal(t, "token required", resp.Error)
}

func TestForbidden(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.Forbidden(c, "insufficient permissions")
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decode(t, w)
	assert.False(t, resp.Success)
	assert.Equal(t, "insufficient permissions", resp.Error)
}

func TestNotFound(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.NotFound(c, "resource not found")
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := decode(t, w)
	assert.False(t, resp.Success)
	assert.Equal(t, "resource not found", resp.Error)
}

func TestConflict(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.Conflict(c, "email already exists")
	})
	assert.Equal(t, http.StatusConflict, w.Code)
	resp := decode(t, w)
	assert.False(t, resp.Success)
	assert.Equal(t, "email already exists", resp.Error)
}

func TestInternalError(t *testing.T) {
	w := callHandler(func(c *gin.Context) {
		response.InternalError(c)
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := decode(t, w)
	assert.False(t, resp.Success)
	assert.Equal(t, "internal server error", resp.Error)
}
