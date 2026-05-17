package api

import "time"

// buildSingleEventQueryResponse assembles a stable single-event response shape.
func buildSingleEventQueryResponse(data any, meta *QueryMeta) *QueryResponse {
	return &QueryResponse{
		Data:      data,
		Events:    data,
		Meta:      meta,
		Timestamp: time.Now().Unix(),
	}
}

// buildPaginatedEventQueryResponse assembles a stable paginated event response shape.
func buildPaginatedEventQueryResponse(data any, limit, offset, total int, meta *QueryMeta) *QueryResponse {
	return &QueryResponse{
		Data:   data,
		Events: data,
		Pagination: &Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
		Meta:      meta,
		Timestamp: time.Now().Unix(),
	}
}
