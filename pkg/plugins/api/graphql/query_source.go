package graphql

func withQuerySourcePosture(result map[string]any, posture string) map[string]any {
	if result == nil || posture == "" {
		return result
	}
	result["querySourcePosture"] = posture
	return result
}

func withQuerySourcePostureList(results []any, posture string) []any {
	if posture == "" || len(results) == 0 {
		return results
	}
	for i := range results {
		if item, ok := results[i].(map[string]any); ok {
			item["querySourcePosture"] = posture
			results[i] = item
		}
	}
	return results
}

func withQuerySourcePostureConnection(connection map[string]any, posture string) map[string]any {
	if connection == nil || posture == "" {
		return connection
	}

	edges, ok := connection["edges"].([]any)
	if !ok || len(edges) == 0 {
		return connection
	}

	for i := range edges {
		edge, ok := edges[i].(map[string]any)
		if !ok {
			continue
		}
		node, ok := edge["node"].(map[string]any)
		if !ok {
			continue
		}
		edge["node"] = withQuerySourcePosture(node, posture)
		edges[i] = edge
	}

	connection["edges"] = edges
	return connection
}
