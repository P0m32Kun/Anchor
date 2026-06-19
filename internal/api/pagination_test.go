package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================
// parsePagination
// ============================================================

func TestParsePagination_AllDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	got := parsePagination(req)
	if got.Page != 1 {
		t.Errorf("Page = %d, want 1", got.Page)
	}
	if got.PageSize != DefaultPageSize {
		t.Errorf("PageSize = %d, want %d", got.PageSize, DefaultPageSize)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{"page only", "page=5", 5, DefaultPageSize},
		{"size only", "page_size=50", 1, 50},
		{"both", "page=3&page_size=100", 3, 100},
		{"page=1 explicit", "page=1", 1, DefaultPageSize},
		{"size=1", "page_size=1", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			got := parsePagination(req)
			if got.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.wantSize)
			}
		})
	}
}

func TestParsePagination_PageClamping(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"zero", "page=0", 1},
		{"negative", "page=-1", 1},
		{"large negative", "page=-999", 1},
		{"invalid string", "page=abc", 1},
		{"empty string", "page=", 1},
		{"float string", "page=3.14", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			got := parsePagination(req)
			if got.Page != tt.want {
				t.Errorf("Page = %d, want %d", got.Page, tt.want)
			}
		})
	}
}

func TestParsePagination_PageSizeClamping(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"zero", "page_size=0", DefaultPageSize},
		{"negative", "page_size=-1", DefaultPageSize},
		{"large negative", "page_size=-999", DefaultPageSize},
		{"invalid string", "page_size=abc", DefaultPageSize},
		{"empty string", "page_size=", DefaultPageSize},
		{"exceeds max", "page_size=9999", MaxPageSize},
		{"exact max", "page_size=1000", MaxPageSize},
		{"just under max", "page_size=999", 999},
		{"one over max", "page_size=1001", MaxPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			got := parsePagination(req)
			if got.PageSize != tt.want {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.want)
			}
		})
	}
}

func TestParsePagination_IgnoresOtherQueryParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?search=foo&page=2&page_size=10&sort=name", nil)
	got := parsePagination(req)
	if got.Page != 2 {
		t.Errorf("Page = %d, want 2", got.Page)
	}
	if got.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", got.PageSize)
	}
}

// ============================================================
// PaginationParams.Offset
// ============================================================

func TestPaginationParams_Offset(t *testing.T) {
	tests := []struct {
		name string
		page int
		size int
		want int
	}{
		{"first page", 1, 20, 0},
		{"second page", 2, 20, 20},
		{"third page", 3, 20, 40},
		{"page 1 size 1", 1, 1, 0},
		{"page 5 size 10", 5, 10, 40},
		{"page 1 size 100", 1, 100, 0},
		{"page 10 size 100", 10, 100, 900},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PaginationParams{Page: tt.page, PageSize: tt.size}
			if got := p.Offset(); got != tt.want {
				t.Errorf("Offset() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ============================================================
// writePaginatedJSON
// ============================================================

func TestWritePaginatedJSON_Basic(t *testing.T) {
	w := httptest.NewRecorder()
	data := []string{"a", "b", "c"}
	pg := PaginationParams{Page: 2, PageSize: 10}

	writePaginatedJSON(w, data, 25, pg)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var result PaginatedResponse[string]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 25 {
		t.Errorf("total = %d, want 25", result.Total)
	}
	if result.Page != 2 {
		t.Errorf("page = %d, want 2", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("page_size = %d, want 10", result.PageSize)
	}
	if len(result.Data) != 3 {
		t.Errorf("len(data) = %d, want 3", len(result.Data))
	}
}

func TestWritePaginatedJSON_EmptyData(t *testing.T) {
	w := httptest.NewRecorder()
	pg := PaginationParams{Page: 1, PageSize: 20}

	writePaginatedJSON(w, []string{}, 0, pg)

	resp := w.Result()
	defer resp.Body.Close()

	var result PaginatedResponse[string]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
	if len(result.Data) != 0 {
		t.Errorf("len(data) = %d, want 0", len(result.Data))
	}
}

func TestWritePaginatedJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	pg := PaginationParams{Page: 1, PageSize: 20}

	writePaginatedJSON[string](w, nil, 0, pg)

	resp := w.Result()
	defer resp.Body.Close()

	var result PaginatedResponse[string]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
}

func TestWritePaginatedJSON_StructData(t *testing.T) {
	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	w := httptest.NewRecorder()
	data := []item{{ID: 1, Name: "foo"}, {ID: 2, Name: "bar"}}
	pg := PaginationParams{Page: 1, PageSize: 50}

	writePaginatedJSON(w, data, 2, pg)

	resp := w.Result()
	defer resp.Body.Close()

	var result PaginatedResponse[item]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(result.Data))
	}
	if result.Data[0].Name != "foo" {
		t.Errorf("data[0].name = %q, want foo", result.Data[0].Name)
	}
}

func TestWritePaginatedJSON_FirstPage(t *testing.T) {
	w := httptest.NewRecorder()
	data := []int{1, 2, 3, 4, 5}
	pg := PaginationParams{Page: 1, PageSize: 5}

	writePaginatedJSON(w, data, 100, pg)

	resp := w.Result()
	defer resp.Body.Close()

	var result PaginatedResponse[int]
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Page != 1 {
		t.Errorf("page = %d, want 1", result.Page)
	}
	if result.Total != 100 {
		t.Errorf("total = %d, want 100", result.Total)
	}
}

func TestWritePaginatedJSON_LastPage(t *testing.T) {
	w := httptest.NewRecorder()
	data := []int{1}
	pg := PaginationParams{Page: 10, PageSize: 10}

	writePaginatedJSON(w, data, 91, pg)

	resp := w.Result()
	defer resp.Body.Close()

	var result PaginatedResponse[int]
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Page != 10 {
		t.Errorf("page = %d, want 10", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("page_size = %d, want 10", result.PageSize)
	}
}

// ============================================================
// PaginationParams struct fields
// ============================================================

func TestPaginationParams_Fields(t *testing.T) {
	p := PaginationParams{Page: 7, PageSize: 25}
	if p.Page != 7 {
		t.Errorf("Page = %d, want 7", p.Page)
	}
	if p.PageSize != 25 {
		t.Errorf("PageSize = %d, want 25", p.PageSize)
	}
}

// ============================================================
// Constants
// ============================================================

func TestPaginationConstants(t *testing.T) {
	if DefaultPageSize != 20 {
		t.Errorf("DefaultPageSize = %d, want 20", DefaultPageSize)
	}
	if MaxPageSize != 1000 {
		t.Errorf("MaxPageSize = %d, want 1000", MaxPageSize)
	}
}

// ============================================================
// PaginatedResponse struct
// ============================================================

func TestPaginatedResponse_JSON(t *testing.T) {
	resp := PaginatedResponse[string]{
		Data:     []string{"x", "y"},
		Total:    10,
		Page:     1,
		PageSize: 2,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got PaginatedResponse[string]
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Total != 10 {
		t.Errorf("total = %d, want 10", got.Total)
	}
	if got.Page != 1 {
		t.Errorf("page = %d, want 1", got.Page)
	}
	if got.PageSize != 2 {
		t.Errorf("page_size = %d, want 2", got.PageSize)
	}
	if len(got.Data) != 2 {
		t.Errorf("len(data) = %d, want 2", len(got.Data))
	}
}
