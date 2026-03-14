package pagination_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/pkg/pagination"
)

func TestEncodeDecode_RoundTrip(t *testing.T) {
	ids := []uint{1, 42, 999, 100000, ^uint(0) >> 1}
	for _, id := range ids {
		encoded := pagination.EncodeCursor(id)
		decoded, err := pagination.DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, id, decoded)
	}
}

func TestDecodeCursor_EmptyString(t *testing.T) {
	id, err := pagination.DecodeCursor("")
	require.NoError(t, err)
	assert.Equal(t, uint(0), id)
}

func TestDecodeCursor_Whitespace(t *testing.T) {
	id, err := pagination.DecodeCursor("  ")
	require.NoError(t, err)
	assert.Equal(t, uint(0), id)
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, err := pagination.DecodeCursor("!!!not-base64!!!")
	assert.Error(t, err)
}

func TestDecodeCursor_ValidBase64ButNotNumber(t *testing.T) {
	// base64url of "abc"
	_, err := pagination.DecodeCursor("YWJj")
	assert.Error(t, err)
}

func TestEncodeCursor_IsURLSafe(t *testing.T) {
	encoded := pagination.EncodeCursor(12345)
	for _, ch := range encoded {
		assert.NotEqual(t, '+', ch, "cursor must not contain '+'")
		assert.NotEqual(t, '/', ch, "cursor must not contain '/'")
		assert.NotEqual(t, '=', ch, "cursor must not contain padding '='")
	}
}
