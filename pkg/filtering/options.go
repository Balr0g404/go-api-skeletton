package filtering

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Order represents the sort direction.
type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

// Options holds parsed sort and filter parameters from a query string.
type Options struct {
	Sort    string
	Order   Order
	Filters map[string]string // field name → value
}

// Allowed defines the whitelists for sort and filter fields of a resource.
type Allowed struct {
	Sort        []string // allowed sort field names
	Filter      []string // allowed filter field names
	DefaultSort string   // fallback when sort param is absent or unknown
}

// Parse reads ?sort=, ?order=, and ?filter[key]=value from a Gin context.
// Unknown sort fields fall back to Allowed.DefaultSort.
// Unknown or empty filter keys are silently dropped.
func Parse(c *gin.Context, allowed Allowed) Options {
	sort := strings.ToLower(c.DefaultQuery("sort", allowed.DefaultSort))
	if !contains(allowed.Sort, sort) {
		sort = allowed.DefaultSort
	}

	order := Order(strings.ToLower(c.DefaultQuery("order", string(OrderAsc))))
	if order != OrderAsc && order != OrderDesc {
		order = OrderAsc
	}

	rawFilters := c.QueryMap("filter")
	filters := make(map[string]string, len(rawFilters))
	for k, v := range rawFilters {
		k = strings.ToLower(k)
		if v != "" && contains(allowed.Filter, k) {
			filters[k] = v
		}
	}

	return Options{Sort: sort, Order: order, Filters: filters}
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}
