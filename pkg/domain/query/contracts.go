package query

//go:generate mockgen -destination=mock_service.go -package=query . Service

import "github.com/rtcdance/chainpulse/pkg/core"

// Request, Result, and Service are type aliases for core.QueryRequest,
// core.QueryResult, and core.QueryService. The concrete types live in
// pkg/core so that service-layer packages can reference them without
// depending on domain/query.
//
// New code should prefer importing from core directly.
type (
	Request = core.QueryRequest
	Result  = core.QueryResult
	Service = core.QueryService
)
