package core

import "time"

const (
	DeploymentMonolithic    = "monolithic"
	DeploymentMicroservice  = "microservice"
	ServiceNameChainPulse   = "chainpulse"
	ComponentQueryService   = "query-service"

	DefaultHTTPTimeout    = 30 * time.Second
	DefaultEventCacheTTL  = 1 * time.Hour
	DefaultStaleThreshold = 5 * time.Minute
)