package models

import "math"

// Pagination holds paging info for list like API.
type Pagination struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
	Pages int `json:"pages"`

	defaultSize int
}

// PaginationOpt represents extra pagination options to apply.
type PaginationOpt func(*Pagination)

// WithDefaultSize set pagination default size.
func WithDefaultSize(size int) PaginationOpt {
	return func(p *Pagination) {
		if size < 0 {
			p.defaultSize = 10
			return
		}
		p.defaultSize = size
	}
}

// NewPagination creates a new Pagination.
func NewPagination(page, size int, opts ...PaginationOpt) *Pagination {
	p := &Pagination{
		Page:        page,
		Size:        size,
		defaultSize: 10,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	p.normalize()
	return p
}

func (p *Pagination) normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Size <= 0 {
		p.Size = p.defaultSize
	}
	if p.Total <= 0 {
		p.Total = 0
	}

	p.Pages = int(math.Ceil(float64(p.Total) / float64(p.Size)))
}

// SetPage update current page index.
func (p *Pagination) SetPage(page int) {
	p.Page = page
	p.normalize()
}

// SetSize update page size.
func (p *Pagination) SetSize(size int) {
	p.Size = size
	p.normalize()
}

// SetTotal update the total records.
func (p *Pagination) SetTotal(total int) {
	p.Total = total
	p.normalize()
}

// Limit returns the page size.
// Attened to be used in SQL statements.
func (p *Pagination) Limit() int {
	p.normalize()
	return p.Size
}

// Offset returns the size of skipped items of current page.
// Attened to be used in SQL statements.
func (p *Pagination) Offset() int {
	p.normalize()
	return (p.Page - 1) * p.Size
}
