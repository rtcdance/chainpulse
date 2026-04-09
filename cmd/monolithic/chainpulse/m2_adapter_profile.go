package main

type monolithicAdapterProfile struct {
	ProfileName              string
	SelectionPosture         string
	ReliabilityHint          string
	IndexingStorageAdapter   string
	QueryRuntimeAdapter      string
	TransportAdapterBoundary string
}

func resolveMonolithicAdapterProfile(mode string, upstreamQueryServices []string) monolithicAdapterProfile {
	switch mode {
	case deploymentModeMicroservice:
		transportBoundary := "runtime-operator-only-gateway-intent"
		selectionPosture := "adapter-profile-partial"
		reliabilityHint := "microservice deployment intent is selected, but monolithic cmd wiring still uses partial compatibility adapters until later M2 slices complete"
		if len(upstreamQueryServices) > 0 {
			transportBoundary = "upstream-query-bridge-gateway-intent"
			selectionPosture = "adapter-profile-bridged"
			reliabilityHint = "microservice deployment intent is selected and the monolithic gateway now exposes a read-only upstream query bridge while compatibility adapters continue to cover the remaining in-process seams"
		}
		return monolithicAdapterProfile{
			ProfileName:              "microservice-target-profile",
			SelectionPosture:         selectionPosture,
			ReliabilityHint:          reliabilityHint,
			IndexingStorageAdapter:   "compatibility-mock-indexing-storage",
			QueryRuntimeAdapter:      "managed-db-runtime-wiring",
			TransportAdapterBoundary: transportBoundary,
		}
	default:
		return monolithicAdapterProfile{
			ProfileName:              "monolithic-runtime-profile",
			SelectionPosture:         "adapter-profile-ready",
			ReliabilityHint:          "monolithic deployment mode is using the current baseline adapter profile for the runnable monolithic path",
			IndexingStorageAdapter:   "monolithic-memory-indexing-storage",
			QueryRuntimeAdapter:      "indexing-backed-query-surface",
			TransportAdapterBoundary: "monolithic-in-process-runtime",
		}
	}
}
