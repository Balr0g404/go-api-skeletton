package filtering_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/pkg/filtering"
)

func init() {
	gin.SetMode(gin.TestMode)
}

var testAllowed = filtering.Allowed{
	Sort:        []string{"id", "created_at", "email"},
	Filter:      []string{"email", "role"},
	DefaultSort: "id",
}

func parseFromURL(url string) filtering.Options {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	c.Request = req
	return filtering.Parse(c, testAllowed)
}

func TestParse_Defaults(t *testing.T) {
	opts := parseFromURL("/users")
	assert.Equal(t, "id", opts.Sort)
	assert.Equal(t, filtering.OrderAsc, opts.Order)
	assert.Empty(t, opts.Filters)
}

func TestParse_ValidSort(t *testing.T) {
	opts := parseFromURL("/users?sort=email")
	assert.Equal(t, "email", opts.Sort)
}

func TestParse_InvalidSortFallsBackToDefault(t *testing.T) {
	opts := parseFromURL("/users?sort=injected_field")
	assert.Equal(t, "id", opts.Sort)
}

func TestParse_SortCaseInsensitive(t *testing.T) {
	opts := parseFromURL("/users?sort=EMAIL")
	assert.Equal(t, "email", opts.Sort)
}

func TestParse_OrderDesc(t *testing.T) {
	opts := parseFromURL("/users?order=desc")
	assert.Equal(t, filtering.OrderDesc, opts.Order)
}

func TestParse_OrderAsc(t *testing.T) {
	opts := parseFromURL("/users?order=asc")
	assert.Equal(t, filtering.OrderAsc, opts.Order)
}

func TestParse_OrderCaseInsensitive(t *testing.T) {
	opts := parseFromURL("/users?order=DESC")
	assert.Equal(t, filtering.OrderDesc, opts.Order)
}

func TestParse_InvalidOrderFallsBackToAsc(t *testing.T) {
	opts := parseFromURL("/users?order=random")
	assert.Equal(t, filtering.OrderAsc, opts.Order)
}

func TestParse_ValidFilter(t *testing.T) {
	opts := parseFromURL("/users?filter[email]=foo@example.com")
	assert.Equal(t, "foo@example.com", opts.Filters["email"])
}

func TestParse_MultipleFilters(t *testing.T) {
	opts := parseFromURL("/users?filter[email]=foo@example.com&filter[role]=admin")
	assert.Equal(t, "foo@example.com", opts.Filters["email"])
	assert.Equal(t, "admin", opts.Filters["role"])
}

func TestParse_UnknownFilterIgnored(t *testing.T) {
	opts := parseFromURL("/users?filter[injected]=value")
	assert.Empty(t, opts.Filters)
}

func TestParse_EmptyFilterValueIgnored(t *testing.T) {
	opts := parseFromURL("/users?filter[email]=")
	assert.Empty(t, opts.Filters)
}

func TestParse_FilterKeysCaseInsensitive(t *testing.T) {
	opts := parseFromURL("/users?filter[EMAIL]=foo@example.com")
	assert.Equal(t, "foo@example.com", opts.Filters["email"])
}

func TestParse_SortAndFilterCombined(t *testing.T) {
	opts := parseFromURL("/users?sort=created_at&order=desc&filter[role]=admin")
	assert.Equal(t, "created_at", opts.Sort)
	assert.Equal(t, filtering.OrderDesc, opts.Order)
	assert.Equal(t, "admin", opts.Filters["role"])
}
