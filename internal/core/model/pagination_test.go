package model_test

import (
	"testing"

	"be-ayaka/internal/core/model"

	"github.com/stretchr/testify/assert"
)

func TestPageRequest_Normalize(t *testing.T) {
	tests := []struct {
		name string
		in   model.PageRequest
		want model.PageRequest
	}{
		{"defaults", model.PageRequest{}, model.PageRequest{Page: 1, PageSize: model.DefaultPageSize}},
		{"clamps size", model.PageRequest{Page: 2, PageSize: 999}, model.PageRequest{Page: 2, PageSize: model.MaxPageSize}},
		{"keeps valid", model.PageRequest{Page: 3, PageSize: 10}, model.PageRequest{Page: 3, PageSize: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.Normalize())
		})
	}
}

func TestPage_Navigation(t *testing.T) {
	req := model.PageRequest{Page: 2, PageSize: 10}
	p := model.NewPage([]int{1, 2, 3}, 25, req)

	assert.Equal(t, 10, req.Offset())
	assert.Equal(t, 3, p.TotalPages())
	assert.True(t, p.HasNext())
	assert.True(t, p.HasPrevious())

	empty := model.NewPage[int](nil, 0, model.PageRequest{Page: 1, PageSize: 10})
	assert.NotNil(t, empty.Items, "items must serialize as [] not null")
	assert.Equal(t, 0, empty.TotalPages())
	assert.False(t, empty.HasNext())
}
