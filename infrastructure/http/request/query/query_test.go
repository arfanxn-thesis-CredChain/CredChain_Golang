package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryRequest_Validate_Valid(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10}
	assert.NoError(t, r.Validate())
}

func TestQueryRequest_Validate_LimitTooLarge(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 200}
	assert.Error(t, r.Validate())
}

func TestQueryRequest_Validate_BadFilterSyntax(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10, Filters: []string{"badfilter"}}
	assert.Error(t, r.Validate())
}

func TestQueryRequest_Validate_StatusOnlyApprovedEquality(t *testing.T) {
	for _, tc := range []struct {
		filter string
		valid  bool
	}{
		{"status=approved", true}, {"status=pending", false}, {"status!=approved", false}, {"status_=", false},
	} {
		err := (&QueryRequest{Filters: []string{tc.filter}}).Validate()
		if tc.valid {
			assert.NoError(t, err, tc.filter)
		} else {
			assert.Error(t, err, tc.filter)
		}
	}
}

func TestQueryRequest_Validate_BadSortSyntax(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10, Sorts: []string{"-"}}
	assert.Error(t, r.Validate())
}

func TestQueryRequest_ToDomain_NilReceiverError(t *testing.T) {
	var r *QueryRequest
	_, err := r.ToDomain()
	assert.Error(t, err)
}

func TestQueryRequest_ToDomain_DefaultsApplied(t *testing.T) {
	r := &QueryRequest{Page: 0, Limit: 0}
	q, err := r.ToDomain()
	assert.NoError(t, err)
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 10, q.Limit)
}

func TestQueryRequest_ToDomain_FiltersParsed(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10, Filters: []string{"name=alice"}}
	q, err := r.ToDomain()
	assert.NoError(t, err)
	assert.Len(t, q.Filters, 1)
	assert.Equal(t, "name", q.Filters[0].Column)
}

func TestQueryRequest_ToDomain_SortsParsed(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10, Sorts: []string{"-created_at"}}
	q, err := r.ToDomain()
	assert.NoError(t, err)
	assert.Len(t, q.Sorts, 1)
	assert.Equal(t, "created_at", q.Sorts[0].Column)
}

func TestQueryRequest_ToDomain_SkipsEmptyEntries(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10, Filters: []string{""}, Sorts: []string{""}}
	q, err := r.ToDomain()
	assert.NoError(t, err)
	assert.Empty(t, q.Filters)
	assert.Empty(t, q.Sorts)
}

func TestQueryRequest_ToDomain_PropagatesParseError(t *testing.T) {
	r := &QueryRequest{Page: 1, Limit: 10, Filters: []string{"badfilter"}}
	_, err := r.ToDomain()
	assert.Error(t, err)
}
