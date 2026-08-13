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
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestPaginateSlice(t *testing.T) {
	tests := []struct {
		name         string
		items        []int
		page         int
		limit        int
		wantLen      int
		wantFirst    int // -1 to skip
		wantTotal    int
		wantTotalPgs int
	}{
		{"FirstPage", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 1, 3, 3, 1, 10, 4},
		{"MiddlePage", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 2, 3, 3, 4, 10, 4},
		{"LastPage", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 4, 3, 1, 10, 10, 4},
		{"BeyondTotal", []int{1, 2, 3}, 5, 3, 0, -1, 3, 1},
		{"Empty", []int{}, 1, 10, 0, -1, 0, 0},
		{"ZeroLimit", []int{1, 2, 3}, 1, 0, 3, 1, 3, 0},
		{"ExactFit", []int{1, 2, 3, 4, 5, 6}, 2, 3, 3, 4, 6, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, meta := PaginateSlice(tc.items, tc.page, tc.limit)
			if len(result) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(result), tc.wantLen)
			}
			if tc.wantFirst >= 0 && len(result) > 0 && result[0] != tc.wantFirst {
				t.Errorf("first = %d, want %d", result[0], tc.wantFirst)
			}
			if meta.Total != tc.wantTotal {
				t.Errorf("total = %d, want %d", meta.Total, tc.wantTotal)
			}
			if meta.TotalPages != tc.wantTotalPgs {
				t.Errorf("totalPages = %d, want %d", meta.TotalPages, tc.wantTotalPgs)
			}
		})
	}
}

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		defLimit  int
		wantPage  int
		wantLimit int
	}{
		{"NoParams", "/api/v2/jobs", 20, 0, 0},
		{"PageOnly", "/api/v2/jobs?page=3", 20, 3, 20},
		{"PageAndLimit", "/api/v2/jobs?page=2&limit=50", 20, 2, 50},
		{"InvalidPage", "/api/v2/jobs?page=abc", 20, 1, 20},
		{"NegativePage", "/api/v2/jobs?page=-1", 20, 1, 20},
		{"LimitOnly_ReturnsAll", "/api/v2/jobs?limit=5", 20, 0, 0},
		{"LimitAboveMax", fmt.Sprintf("/api/v2/jobs?page=1&limit=%d", maxPageSize+100), 20, 1, maxPageSize},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.query, nil)
			page, limit := ParsePaginationParams(req, tc.defLimit)

			if page != tc.wantPage {
				t.Errorf("page = %d, want %d", page, tc.wantPage)
			}
			if limit != tc.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tc.wantLimit)
			}
		})
	}
}
