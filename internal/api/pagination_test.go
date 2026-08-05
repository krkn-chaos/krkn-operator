/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"net/http/httptest"
	"testing"
)

func TestPaginateSlice_FirstPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result, meta := PaginateSlice(items, 1, 3)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", result)
	}
	if meta.Total != 10 {
		t.Errorf("expected total 10, got %d", meta.Total)
	}
	if meta.TotalPages != 4 {
		t.Errorf("expected 4 pages, got %d", meta.TotalPages)
	}
	if meta.Page != 1 {
		t.Errorf("expected page 1, got %d", meta.Page)
	}
}

func TestPaginateSlice_MiddlePage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result, meta := PaginateSlice(items, 2, 3)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != 4 || result[1] != 5 || result[2] != 6 {
		t.Errorf("expected [4,5,6], got %v", result)
	}
	if meta.Total != 10 {
		t.Errorf("expected total 10, got %d", meta.Total)
	}
}

func TestPaginateSlice_LastPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result, meta := PaginateSlice(items, 4, 3)

	if len(result) != 1 {
		t.Errorf("expected 1 item on last page, got %d", len(result))
	}
	if result[0] != 10 {
		t.Errorf("expected [10], got %v", result)
	}
	if meta.TotalPages != 4 {
		t.Errorf("expected 4 pages, got %d", meta.TotalPages)
	}
}

func TestPaginateSlice_BeyondTotal(t *testing.T) {
	items := []int{1, 2, 3}
	result, meta := PaginateSlice(items, 5, 3)

	if len(result) != 0 {
		t.Errorf("expected empty slice for page beyond total, got %d items", len(result))
	}
	if meta.Total != 3 {
		t.Errorf("expected total 3, got %d", meta.Total)
	}
}

func TestPaginateSlice_Empty(t *testing.T) {
	items := []int{}
	result, meta := PaginateSlice(items, 1, 10)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
	if meta.Total != 0 {
		t.Errorf("expected total 0, got %d", meta.Total)
	}
	if meta.TotalPages != 0 {
		t.Errorf("expected 0 pages, got %d", meta.TotalPages)
	}
}

func TestPaginateSlice_ZeroLimit(t *testing.T) {
	items := []int{1, 2, 3}
	result, _ := PaginateSlice(items, 1, 0)

	if len(result) != 3 {
		t.Errorf("expected all items with zero limit, got %d", len(result))
	}
}

func TestPaginateSlice_ExactFit(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	result, meta := PaginateSlice(items, 2, 3)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if meta.TotalPages != 2 {
		t.Errorf("expected 2 pages for exact fit, got %d", meta.TotalPages)
	}
}

func TestParsePaginationParams_NoPagination(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v2/jobs", nil)
	page, limit := ParsePaginationParams(req, 20)

	if page != 0 || limit != 0 {
		t.Errorf("expected (0,0) when no params, got (%d,%d)", page, limit)
	}
}

func TestParsePaginationParams_WithPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v2/jobs?page=3", nil)
	page, limit := ParsePaginationParams(req, 20)

	if page != 3 {
		t.Errorf("expected page 3, got %d", page)
	}
	if limit != 20 {
		t.Errorf("expected default limit 20, got %d", limit)
	}
}

func TestParsePaginationParams_WithBoth(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v2/jobs?page=2&limit=50", nil)
	page, limit := ParsePaginationParams(req, 20)

	if page != 2 {
		t.Errorf("expected page 2, got %d", page)
	}
	if limit != 50 {
		t.Errorf("expected limit 50, got %d", limit)
	}
}

func TestParsePaginationParams_InvalidPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v2/jobs?page=abc", nil)
	page, limit := ParsePaginationParams(req, 20)

	if page != 1 {
		t.Errorf("expected default page 1 for invalid input, got %d", page)
	}
	if limit != 20 {
		t.Errorf("expected default limit 20, got %d", limit)
	}
}

func TestParsePaginationParams_NegativePage(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v2/jobs?page=-1", nil)
	page, _ := ParsePaginationParams(req, 20)

	if page != 1 {
		t.Errorf("expected default page 1 for negative input, got %d", page)
	}
}
