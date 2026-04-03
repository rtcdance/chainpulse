package query

import (
	appquery "chainpulse/pkg/application/query"
	domainquery "chainpulse/pkg/domain/query"
	legacyquery "chainpulse/pkg/services/query"
)

// NewDomainServiceFromLegacy adapts legacy query service to domain contract.
func NewDomainServiceFromLegacy(legacy legacyquery.QueryService) domainquery.Service {
	return appquery.NewLegacyFacade(legacy)
}
