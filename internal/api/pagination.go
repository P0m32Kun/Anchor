package api

import (
	"net/http"
	"strconv"
)

type PaginationParams struct {
	Page     int
	PageSize int
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 1000
)

func parsePagination(r *http.Request) PaginationParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return PaginationParams{Page: page, PageSize: pageSize}
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type PaginatedResponse[T any] struct {
	Data     []T `json:"data"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func writePaginatedJSON[T any](w http.ResponseWriter, data []T, total int, page PaginationParams) {
	writeJSON(w, http.StatusOK, PaginatedResponse[T]{
		Data:     data,
		Total:    total,
		Page:     page.Page,
		PageSize: page.PageSize,
	})
}
