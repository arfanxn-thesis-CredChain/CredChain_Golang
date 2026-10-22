package query

import (
	"fmt"

	domainQuery "CredChain_Golang/domain/query"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type QueryRequest struct {
	Page     int      `json:"page" form:"page"`
	Limit    int      `json:"limit" form:"limit"`
	Search   string   `json:"search" form:"search"`
	Filters  []string `json:"filters" form:"filters"`
	Sorts    []string `json:"sorts" form:"sorts"`
	Includes []string `json:"includes" form:"includes"`
}

func (r *QueryRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Page, validation.Min(1)),
		validation.Field(&r.Limit, validation.Min(1), validation.Max(100)),
		validation.Field(&r.Filters, validation.Each(validation.By(validateFilter))),
		validation.Field(&r.Sorts, validation.Each(validation.By(validateSort))),
	)
}

func validateFilter(value interface{}) error {
	filter, ok := value.(string)
	if !ok {
		return validation.NewError("validation_string_required", "must be a string")
	}
	if filter == "" {
		return nil
	}
	column, op, values, err := parseFilterString(filter)
	if err != nil {
		return validation.NewError("validation_syntax_invalid", "has invalid syntax")
	}
	if column == "status" && (op != domainQuery.OperatorEqual || len(values) != 1 || values[0] != "approved") {
		return validation.NewError("validation_syntax_invalid", "status supports only =approved")
	}
	return nil
}

func validateSort(value interface{}) error {
	sort, ok := value.(string)
	if !ok {
		return validation.NewError("validation_string_required", "must be a string")
	}
	if sort == "" {
		return nil
	}
	_, _, err := parseSortString(sort)
	if err != nil {
		return validation.NewError("validation_syntax_invalid", "has invalid syntax")
	}
	return nil
}

func (r *QueryRequest) ToDomain() (*domainQuery.Query, error) {
	if r == nil {
		return nil, fmt.Errorf("QueryRequest is nil")
	}

	query := &domainQuery.Query{
		Page:     r.Page,
		Limit:    r.Limit,
		Search:   r.Search,
		Filters:  []domainQuery.Filter{},
		Sorts:    []domainQuery.Sort{},
		Includes: r.Includes,
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 {
		query.Limit = 10
	}

	for _, f := range r.Filters {
		if f == "" {
			continue
		}
		column, operator, values, err := parseFilterString(f)
		if err != nil {
			return nil, fmt.Errorf("parse filter %s: %w", f, err)
		}
		query.Filters = append(query.Filters, domainQuery.NewFilter(column, operator, values...))
	}

	for _, s := range r.Sorts {
		if s == "" {
			continue
		}
		column, order, err := parseSortString(s)
		if err != nil {
			return nil, fmt.Errorf("parse sort %s: %w", s, err)
		}
		query.Sorts = append(query.Sorts, domainQuery.NewSort(column, order))
	}

	return query, nil
}
