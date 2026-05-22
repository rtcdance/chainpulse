package query

import "github.com/rtcdance/chainpulse/pkg/services/query/qerrors"

// Backward-compatible shims for types moved to query/qerrors.

var (
	ErrQueryOptimizationFailed          = qerrors.ErrQueryOptimizationFailed
	ErrInvalidQuery                     = qerrors.ErrInvalidQuery
	ErrQueryPlanAnalysisFailed          = qerrors.ErrQueryPlanAnalysisFailed
	ErrIndexNotFound                    = qerrors.ErrIndexNotFound
	ErrIndexAlreadyExists               = qerrors.ErrIndexAlreadyExists
	ErrInvalidIndexParameters           = qerrors.ErrInvalidIndexParameters
	ErrIndexCreationFailed              = qerrors.ErrIndexCreationFailed
	ErrIndexDeletionFailed              = qerrors.ErrIndexDeletionFailed
	ErrIndexRebuildFailed               = qerrors.ErrIndexRebuildFailed
	ErrIndexFragmentationAnalysisFailed = qerrors.ErrIndexFragmentationAnalysisFailed
	ErrStatisticsCollectionFailed       = qerrors.ErrStatisticsCollectionFailed
	ErrCacheOperationFailed             = qerrors.ErrCacheOperationFailed
	ErrPaginationFailed                 = qerrors.ErrPaginationFailed
	ErrFilterOptimizationFailed         = qerrors.ErrFilterOptimizationFailed
)

type ErrorType = qerrors.Type
type ErrorClassifier = qerrors.Classifier

const (
	ErrorTypeTransient = qerrors.TypeTransient
	ErrorTypePermanent = qerrors.TypePermanent
	ErrorTypeCritical  = qerrors.TypeCritical
	ErrorTypeUnknown   = qerrors.TypeUnknown
)

func NewErrorClassifier() *ErrorClassifier {
	return qerrors.NewClassifier()
}
