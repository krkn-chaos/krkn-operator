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
	"math"
	"net/http"
	"strconv"
)

// PaginateSlice returns the items for the given page and pagination metadata.
// page is 1-based. If page is beyond the total, an empty slice is returned.
func PaginateSlice[T any](items []T, page, limit int) ([]T, PaginationMeta) {
	total := len(items)
	totalPages := 0
	if limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	meta := PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	if limit <= 0 || page <= 0 {
		return items, meta
	}

	offset := (page - 1) * limit
	if offset >= total {
		return []T{}, meta
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return items[offset:end], meta
}

const maxPageSize = 500

// ParsePaginationParams parses page and limit query parameters from the request.
// Returns (0, 0) if the page param is absent (meaning "return all").
// The limit param is only read when page is present.
func ParsePaginationParams(r *http.Request, defaultLimit int) (page, limit int) {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 0, 0
	}

	page = 1
	limit = defaultLimit

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if limit > maxPageSize {
		limit = maxPageSize
	}

	return page, limit
}
