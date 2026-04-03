package api

type eventQueryMetaInput struct {
	Source               string
	QuerySourcePosture   string
	QueryPath            string
	FallbackUsed         bool
	MetadataCompleteness string
	MetadataAttachedCount int
	MetadataMissingCount  int
	ResultCount          int
}

func buildEventQueryMetaFromInput(input eventQueryMetaInput) *QueryMeta {
	sourcePosture := input.QuerySourcePosture
	if sourcePosture == "" {
		sourcePosture = classifyEventQuerySourcePosture(input.Source, input.FallbackUsed, false)
	}

	coveragePosture := classifyEventQueryMetadataCoveragePosture(input.ResultCount, input.MetadataAttachedCount)
	consistency := classifyEventQueryConsistencyPosture(input.Source, input.QueryPath, input.FallbackUsed, coveragePosture)

	return &QueryMeta{
		Source:                 input.Source,
		QuerySourcePosture:     sourcePosture,
		QueryPath:              input.QueryPath,
		FallbackUsed:           input.FallbackUsed,
		MetadataCompleteness:   input.MetadataCompleteness,
		MetadataCoveragePosture: coveragePosture,
		ConsistencyPosture:     consistency,
		QueryReliabilityHint:   buildEventQueryReliabilityHint(sourcePosture, consistency),
		QueryExecutionSummary:  buildEventQueryExecutionSummary(input.Source, input.QueryPath, input.FallbackUsed, coveragePosture),
		MetadataAttachedCount:  input.MetadataAttachedCount,
		MetadataMissingCount:   input.MetadataMissingCount,
		ResultCount:            input.ResultCount,
	}
}
