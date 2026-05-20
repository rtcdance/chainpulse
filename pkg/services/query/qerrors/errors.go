// Package qerrors provides error types and classification for the query service.
package qerrors

import "errors"

var (
	ErrQueryOptimizationFailed       = errors.New("query optimization failed")
	ErrInvalidQuery                  = errors.New("invalid query")
	ErrQueryPlanAnalysisFailed       = errors.New("query plan analysis failed")
	ErrIndexNotFound                 = errors.New("index not found")
	ErrIndexAlreadyExists            = errors.New("index already exists")
	ErrInvalidIndexParameters        = errors.New("invalid index parameters")
	ErrIndexCreationFailed           = errors.New("index creation failed")
	ErrIndexDeletionFailed           = errors.New("index deletion failed")
	ErrIndexRebuildFailed            = errors.New("index rebuild failed")
	ErrIndexFragmentationAnalysisFailed = errors.New("index fragmentation analysis failed")
	ErrStatisticsCollectionFailed    = errors.New("statistics collection failed")
	ErrCacheOperationFailed          = errors.New("cache operation failed")
	ErrPaginationFailed              = errors.New("pagination failed")
	ErrFilterOptimizationFailed      = errors.New("filter optimization failed")
)