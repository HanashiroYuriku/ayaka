package model

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PageRequest describes which page of a collection to load.
type PageRequest struct {
	Page     int
	PageSize int
}

// Normalize applies defaults and bounds so repositories can trust the values.
func (p PageRequest) Normalize() PageRequest {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
	return p
}

func (p PageRequest) Offset() int { return (p.Page - 1) * p.PageSize }
func (p PageRequest) Limit() int  { return p.PageSize }

// Page is a single page of results plus the information needed to navigate.
type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}

func NewPage[T any](items []T, total int64, req PageRequest) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize}
}

func (p Page[T]) TotalPages() int {
	if p.PageSize <= 0 {
		return 0
	}
	return int((p.Total + int64(p.PageSize) - 1) / int64(p.PageSize))
}

func (p Page[T]) HasNext() bool     { return p.Page < p.TotalPages() }
func (p Page[T]) HasPrevious() bool { return p.Page > 1 }

// MapPage converts the items of a page while preserving its metadata.
func MapPage[T, U any](p Page[T], fn func(T) U) Page[U] {
	out := make([]U, len(p.Items))
	for i, item := range p.Items {
		out[i] = fn(item)
	}
	return Page[U]{Items: out, Total: p.Total, Page: p.Page, PageSize: p.PageSize}
}
