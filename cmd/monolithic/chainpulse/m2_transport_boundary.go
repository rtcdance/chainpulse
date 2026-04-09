package main

type monolithicTransportBoundaryStatus struct {
	Posture string
	Hint    string
}

func classifyMonolithicTransportBoundary(
	boundary string,
	gatewaySurfaceMode string,
	configured int,
	attached int,
	available int,
) monolithicTransportBoundaryStatus {
	switch boundary {
	case "monolithic-in-process-runtime":
		return monolithicTransportBoundaryStatus{
			Posture: "transport-boundary-in-process-ready",
			Hint:    "monolithic deployment mode keeps query and runtime transport ownership in-process for the current runnable baseline",
		}
	case "runtime-operator-only-gateway-intent":
		return monolithicTransportBoundaryStatus{
			Posture: "transport-boundary-runtime-operator-only",
			Hint:    "microservice deployment intent keeps the monolithic gateway on runtime and operator routes only until cross-service query transport is explicitly bridged",
		}
	case "upstream-query-bridge-gateway-intent":
		switch {
		case configured == 0:
			return monolithicTransportBoundaryStatus{
				Posture: "transport-boundary-bridge-unconfigured",
				Hint:    "microservice deployment intent selected an upstream query bridge boundary, but no upstream query services are configured",
			}
		case attached == 0:
			return monolithicTransportBoundaryStatus{
				Posture: "transport-boundary-bridge-unattached",
				Hint:    "upstream query services are configured, but the monolithic gateway has not attached them to the shared transport boundary yet",
			}
		case available == 0:
			return monolithicTransportBoundaryStatus{
				Posture: "transport-boundary-bridge-unavailable",
				Hint:    "the monolithic gateway is using an upstream query bridge boundary, but no attached upstream is currently healthy",
			}
		case available < attached:
			return monolithicTransportBoundaryStatus{
				Posture: "transport-boundary-bridge-degraded",
				Hint:    "the monolithic gateway has a partially healthy upstream query bridge; treat the transport boundary as degraded until all attached upstreams recover",
			}
		default:
			return monolithicTransportBoundaryStatus{
				Posture: "transport-boundary-bridge-ready",
				Hint:    "microservice deployment intent is now backed by a healthy read-only upstream query bridge while runtime and operator routes remain locally owned",
			}
		}
	}

	if gatewaySurfaceMode == "runtime-operator-only" {
		return monolithicTransportBoundaryStatus{
			Posture: "transport-boundary-runtime-operator-only",
			Hint:    "the monolithic gateway currently exposes only runtime and operator routes",
		}
	}

	return monolithicTransportBoundaryStatus{
		Posture: "transport-boundary-unclassified",
		Hint:    "the monolithic transport boundary is not yet classified; verify gateway surface and upstream query bridge wiring before treating deployment switching as ready",
	}
}
