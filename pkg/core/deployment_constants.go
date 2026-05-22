package core

import "time"

const (
	DeploymentModeMonolithic   = "monolithic"
	DeploymentModeMicroservice = "microservice"
)

const DefaultTimeout = 30 * time.Second
