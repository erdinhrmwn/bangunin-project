// Package pagination provides page-based list pagination (default 20,
// max 100 per NFR-4). Cursor-based pagination is a separate concern for
// later phases' large catalog listings — not unified with this helper.
package pagination

import "strconv"

const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type Params struct {
	Page    int
	PerPage int
}

// Parse parses page/per_page query params, defaulting and clamping
// invalid or out-of-range values rather than erroring.
func Parse(pageStr, perPageStr string) Params {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Params{Page: page, PerPage: perPage}
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}
