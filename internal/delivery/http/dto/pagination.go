package dto

import "be-ayaka/internal/core/model"

// PaginationQuery is bound from ?page=&pageSize=. Both are optional.
type PaginationQuery struct {
	Page     int `query:"page" validate:"omitempty,min=1" example:"1"`
	PageSize int `query:"pageSize" validate:"omitempty,min=1,max=100" example:"20"`
}

func (q PaginationQuery) ToPageRequest() model.PageRequest {
	return model.PageRequest{Page: q.Page, PageSize: q.PageSize}.Normalize()
}

// PaginationMeta is returned in the `meta` field of paginated responses.
type PaginationMeta struct {
	TotalRecords int64 `json:"totalRecords" example:"42"`
	TotalPages   int   `json:"totalPages" example:"3"`
	CurrentPage  int   `json:"currentPage" example:"1"`
	PageSize     int   `json:"pageSize" example:"20"`
	HasNext      bool  `json:"hasNext" example:"true"`
	HasPrevious  bool  `json:"hasPrevious" example:"false"`
}

// NewPaginated converts a core page into the response items and meta.
func NewPaginated[T, U any](p *model.Page[T], mapItem func(T) U) ([]U, PaginationMeta) {
	items := make([]U, len(p.Items))
	for i, it := range p.Items {
		items[i] = mapItem(it)
	}
	return items, PaginationMeta{
		TotalRecords: p.Total,
		TotalPages:   p.TotalPages(),
		CurrentPage:  p.Page,
		PageSize:     p.PageSize,
		HasNext:      p.HasNext(),
		HasPrevious:  p.HasPrevious(),
	}
}
