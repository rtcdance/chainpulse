package query

import "errors"

// Query optimization errors
var (
	// ErrQueryOptimizationFailed is returned when query optimization fails
	ErrQueryOptimizationFailed = errors.New("query optimization failed")

	// ErrInvalidQuery is returned when a query is invalid
	ErrInvalidQuery = errors.New("invalid query")

	// ErrQueryPlanAnalysisFailed is returned when query plan analysis fails
	ErrQueryPlanAnalysisFailed = errors.New("query plan analysis failed")

	// ErrIndexNotFound is returned when an index is not found
	ErrIndexNotFound = errors.New("index not found")

	// ErrIndexAlreadyExists is returned when an index already exists
	ErrIndexAlreadyExists = errors.New("index already exists")

	// ErrInvalidIndexParameters is returned when index parameters are invalid
	ErrInvalidIndexParameters = errors.New("invalid index parameters")

	// ErrIndexCreationFailed is returned when index creation fails
	ErrIndexCreationFailed = errors.New("index creation failed")

	// ErrIndexDeletionFailed is returned when index deletion fails
	ErrIndexDeletionFailed = errors.New("index deletion failed")

	// ErrIndexRebuildFailed is returned when index rebuild fails
	ErrIndexRebuildFailed = errors.New("index rebuild failed")

	// ErrIndexFragmentationAnalysisFailed is returned when fragmentation analysis fails
	ErrIndexFragmentationAnalysisFailed = errors.New("index fragmentation analysis failed")

	// ErrStatisticsCollectionFailed is returned when statistics collection fails
	ErrStatisticsCollectionFailed = errors.New("statistics collection failed")

	// ErrCacheOperationFailed is returned when a cache operation fails
	ErrCacheOperationFailed = errors.New("cache operation failed")

	// ErrPaginationFailed is returned when pagination fails
	ErrPaginationFailed = errors.New("pagination failed")

	// ErrFilterOptimizationFailed is returned when filter optimization fails
	ErrFilterOptimizationFailed = errors.New("filter optimization failed")
)
