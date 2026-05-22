package reliability

import "context"

type LeaderRole string

const (
	LeaderRolePuller       LeaderRole = "puller"
	LeaderRoleIndexer      LeaderRole = "indexer"
	LeaderRoleConsolidator LeaderRole = "consolidator"
)

type LeaderElector interface {
	Campaign(ctx context.Context, role LeaderRole, instanceID string) (<-chan bool, error)
	Resign(ctx context.Context) error
	IsLeader() bool
	Observe(ctx context.Context, role LeaderRole) <-chan string
}
