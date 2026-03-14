package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// EncodeCursor encodes a uint ID into an opaque base64url cursor string.
func EncodeCursor(id uint) string {
	raw := fmt.Sprintf("%d", id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor string back to a uint ID.
// Returns 0 and no error when the cursor is empty (first page).
func DecodeCursor(cursor string) (uint, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	id, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	return uint(id), nil
}
