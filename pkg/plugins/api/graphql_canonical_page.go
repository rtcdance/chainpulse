package api

import (
	"context"
	"fmt"

	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

func (h *GraphQLHandler) canonicalPage(ctx context.Context, chain int, first int, after string) (map[string]any, error) {
	filter := map[string]any{}
	if chain > 0 {
		filter["chainId"] = fmt.Sprint(chain)
	}
	result, err := h.queryService.Query(ctx, &domainquery.Request{Filter: filter, Limit: int64(first), Cursor: after})
	if err != nil {
		return nil, err
	}
	edges := make([]any, len(result.Events))
	for i := range result.Events {
		cursor := ""
		if i < len(result.Cursors) {
			cursor = result.Cursors[i]
		}
		edges[i] = map[string]any{"node": eventToMap(&result.Events[i]), "cursor": cursor}
	}
	start, end := "", ""
	if len(result.Cursors) > 0 {
		start = result.Cursors[0]
		end = result.Cursors[len(result.Cursors)-1]
	}
	return map[string]any{"edges": edges, "total": result.Total, "pageInfo": map[string]any{"hasNextPage": result.NextCursor != "", "hasPreviousPage": after != "", "startCursor": start, "endCursor": end}}, nil
}
