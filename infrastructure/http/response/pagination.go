package response

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pagination structures the paginated response identically to the alphabetical JS format requirement.
type Pagination[T any] struct {
	FirstPageURL *string `json:"first_page_url"`
	From         int     `json:"from"`
	Items        []T     `json:"items"`
	LastPage     int     `json:"last_page"`
	LastPageURL  *string `json:"last_page_url"`
	Limit        int     `json:"limit"`
	NextPageURL  *string `json:"next_page_url"`
	Page         int     `json:"page"`
	PrevPageURL  *string `json:"prev_page_url"`
	To           int     `json:"to"`
	Total        int     `json:"total"`
}

// NewPaginationFromContext constructs a Pagination based on items, total, and gin
// context, deriving page and limit from the query string.
//
// Only correct when the handler used those same values. A handler that applies
// its own default (see the lookup endpoints, which page at 50-100) must call
// NewPagination with the effective values instead, or the envelope reports a
// limit the rows do not match and last_page comes out wrong.
func NewPaginationFromContext[T any](c *gin.Context, items []T, total int) Pagination[T] {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	return NewPaginationWithPageLimit(c, items, total, page, limit)
}

// NewPaginationWithPageLimit constructs a Pagination from the page and limit the
// handler actually applied, rather than re-reading them from the query string.
func NewPaginationWithPageLimit[T any](c *gin.Context, items []T, total, page, limit int) Pagination[T] {
	if items == nil {
		items = make([]T, 0)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	lastPage := max(int(math.Ceil(float64(total)/float64(limit))), 1)

	from := (page-1)*limit + 1
	to := page * limit
	if total == 0 {
		from = 0
		to = 0
	} else if to > total {
		to = total
	}

	baseUrl := fmt.Sprintf("%s://%s%s", "http", c.Request.Host, c.Request.URL.Path)

	buildURL := func(targetPage int) *string {
		q := c.Request.URL.Query()
		q.Set("page", fmt.Sprintf("%d", targetPage))
		q.Set("limit", fmt.Sprintf("%d", limit))
		urlStr := fmt.Sprintf("%s?%s", baseUrl, q.Encode())
		return &urlStr
	}

	firstPageUrl := buildURL(1)
	lastPageUrl := buildURL(lastPage)

	var prevPageUrl *string
	if page > 1 {
		prevPageUrl = buildURL(page - 1)
	}

	var nextPageUrl *string
	if page < lastPage {
		nextPageUrl = buildURL(page + 1)
	}

	return Pagination[T]{
		FirstPageURL: firstPageUrl,
		From:         from,
		Items:        items,
		LastPage:     lastPage,
		LastPageURL:  lastPageUrl,
		Limit:        limit,
		NextPageURL:  nextPageUrl,
		Page:         page,
		PrevPageURL:  prevPageUrl,
		To:           to,
		Total:        total,
	}
}
