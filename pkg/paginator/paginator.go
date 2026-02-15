package paginator

import "math"

const (
	// DefaultPage is the default page number when invalid page is provided.
	DefaultPage = 1
	// DefaultLimit is the default number of items per page when invalid limit is provided.
	DefaultLimit = 15
	// MaxLimit is the maximum number of items per page to prevent excessive queries.
	MaxLimit = 100
)

// PaginateQuery contains pagination parameters for a request.
type PaginateQuery struct {
	Page  int   `json:"page" form:"page"`   // Page number (1-indexed)
	Limit int64 `json:"limit" form:"limit"` // Number of items per page
}

// Adjust normalizes the pagination parameters to valid values.
// Sets defaults if values are invalid and enforces maximum limit.
func (p *PaginateQuery) Adjust() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}

	if p.Limit < 1 {
		p.Limit = DefaultLimit
	} else if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
}

// Offset calculates the database offset for the current page.
// Returns the number of items to skip before returning results.
func (p *PaginateQuery) Offset() int64 {
	return int64((p.Page - 1)) * p.Limit
}

// Paginator contains pagination metadata for a query result.
type Paginator struct {
	Total       int64 `json:"total"`        // Total number of items across all pages
	Count       int64 `json:"count"`        // Number of items in current page
	PerPage     int64 `json:"per_page"`     // Number of items per page
	CurrentPage int   `json:"current_page"` // Current page number (1-indexed)
}

// TotalPages calculates the total number of pages based on total items and items per page.
func (p Paginator) TotalPages() int {
	if p.Total == 0 || p.PerPage == 0 {
		return 0
	}

	return int(math.Ceil(float64(p.Total) / float64(p.PerPage)))
}

// HasNextPage checks if there is a next page available.
func (p Paginator) HasNextPage() bool {
	return p.CurrentPage < p.TotalPages()
}

// HasPreviousPage checks if there is a previous page available.
func (p Paginator) HasPreviousPage() bool {
	return p.CurrentPage > 1
}

// ToResponse converts the paginator to a response format with additional calculated fields.
func (p Paginator) ToResponse() PaginatorResponse {
	return PaginatorResponse{
		Total:       p.Total,
		Count:       p.Count,
		PerPage:     p.PerPage,
		CurrentPage: p.CurrentPage,
		TotalPages:  p.TotalPages(),
		HasNext:     p.HasNextPage(),
		HasPrev:     p.HasPreviousPage(),
	}
}

// PaginatorResponse is the response format for pagination metadata.
// Includes calculated fields for convenience.
type PaginatorResponse struct {
	Total       int64 `json:"total"`        // Total number of items across all pages
	Count       int64 `json:"count"`        // Number of items in current page
	PerPage     int64 `json:"per_page"`     // Number of items per page
	CurrentPage int   `json:"current_page"` // Current page number (1-indexed)
	TotalPages  int   `json:"total_pages"`  // Total number of pages
	HasNext     bool  `json:"has_next"`     // Whether there is a next page
	HasPrev     bool  `json:"has_prev"`     // Whether there is a previous page
}

// ToPaginator converts a response back to a paginator (useful for deserialization).
func (p PaginatorResponse) ToPaginator() Paginator {
	return Paginator{
		Total:       p.Total,
		Count:       p.Count,
		PerPage:     p.PerPage,
		CurrentPage: p.CurrentPage,
	}
}
