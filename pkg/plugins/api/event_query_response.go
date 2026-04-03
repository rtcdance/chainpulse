package api

import "time"

// buildSingleEventQueryResponse assembles a stable single-event response shape.
func buildSingleEventQueryResponse(data interface{}, meta *QueryMeta) *QueryResponse {
	return &QueryResponse{
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().Unix(),
	}
}

// buildPaginatedEventQueryResponse assembles a stable paginated event response shape.
func buildPaginatedEventQueryResponse(data interface{}, limit, offset, total int, meta *QueryMeta) *QueryResponse {
	return &QueryResponse{
		Data: data,
		Pagination: &Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
		Meta:      meta,
		Timestamp: time.Now().Unix(),
	}
}
