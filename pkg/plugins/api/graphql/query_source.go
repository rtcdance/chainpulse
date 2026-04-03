package graphql

func withQuerySourcePosture(result map[string]interface{}, posture string) map[string]interface{} {
	if result == nil || posture == "" {
		return result
	}
	result["querySourcePosture"] = posture
	return result
}

func withQuerySourcePostureList(results []interface{}, posture string) []interface{} {
	if posture == "" || len(results) == 0 {
		return results
	}
	for i := range results {
		if item, ok := results[i].(map[string]interface{}); ok {
			item["querySourcePosture"] = posture
			results[i] = item
		}
	}
	return results
}

func withQuerySourcePostureConnection(connection map[string]interface{}, posture string) map[string]interface{} {
	if connection == nil || posture == "" {
		return connection
	}

	edges, ok := connection["edges"].([]interface{})
	if !ok || len(edges) == 0 {
		return connection
	}

	for i := range edges {
		edge, ok := edges[i].(map[string]interface{})
		if !ok {
			continue
		}
		node, ok := edge["node"].(map[string]interface{})
		if !ok {
			continue
		}
		edge["node"] = withQuerySourcePosture(node, posture)
		edges[i] = edge
	}

	connection["edges"] = edges
	return connection
}
