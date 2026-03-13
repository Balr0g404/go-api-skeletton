package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
)

func newManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret", 1, 24)
}

func TestGenerateTokenPair_Success(t *testing.T) {
	m := newManager()
	pair, err := m.GenerateTokenPair(42, "user@example.com", "user")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, pair.RefreshToken)
}

func TestValidateToken_AccessToken(t *testing.T) {
	m := newManager()
	pair, err := m.GenerateTokenPair(42, "user@example.com", "user")
	require.NoError(t, err)

	claims, err := m.ValidateToken(pair.AccessToken, auth.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "user", claims.Role)
	assert.Equal(t, auth.AccessToken, claims.Type)
}

func TestValidateToken_RefreshToken(t *testing.T) {
	m := newManager()
	pair, err := m.GenerateTokenPair(1, "admin@example.com", "admin")
	require.NoError(t, err)

	claims, err := m.ValidateToken(pair.RefreshToken, auth.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, auth.RefreshToken, claims.Type)
}

func TestValidateToken_WrongType(t *testing.T) {
	m := newManager()
	pair, err := m.GenerateTokenPair(1, "user@example.com", "user")
	require.NoError(t, err)

	_, err = m.ValidateToken(pair.AccessToken, auth.RefreshToken)
	assert.ErrorContains(t, err, "invalid token type")
}

func TestValidateToken_InvalidToken(t *testing.T) {
	m := newManager()
	_, err := m.ValidateToken("not.a.valid.token", auth.AccessToken)
	assert.Error(t, err)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	m1 := auth.NewJWTManager("secret-one", 1, 24)
	m2 := auth.NewJWTManager("secret-two", 1, 24)

	pair, err := m1.GenerateTokenPair(1, "user@example.com", "user")
	require.NoError(t, err)

	_, err = m2.ValidateToken(pair.AccessToken, auth.AccessToken)
	assert.Error(t, err)
}

func TestValidateToken_Expired(t *testing.T) {
	claims := &auth.Claims{
		UserID: 1,
		Email:  "user@example.com",
		Role:   "user",
		Type:   auth.AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	m := newManager()
	_, err = m.ValidateToken(signed, auth.AccessToken)
	assert.Error(t, err)
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	claims := &auth.Claims{
		UserID: 1,
		Email:  "user@example.com",
		Role:   "user",
		Type:   auth.AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	require.NoError(t, err)

	m := newManager()
	_, err = m.ValidateToken(signed, auth.AccessToken)
	assert.Error(t, err)
}
