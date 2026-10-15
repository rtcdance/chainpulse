package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/stretchr/testify/require"
)

type failingCloseDB struct {
	fakeDBManager
	closed bool
	err    error
}

func (d *failingCloseDB) Close(context.Context) error { d.closed = true; return d.err }

type failingStopQuery struct {
	fakeQueryRuntimeService
	stopped bool
	err     error
}

func (q *failingStopQuery) Stop(context.Context) error { q.stopped = true; return q.err }

func TestRuntimeCloseReportsErrorsAndStillClosesRemainingOwners(t *testing.T) {
	t.Parallel()
	dbErr := errors.New("database close failed")
	queryErr := errors.New("query close failed")
	db := &failingCloseDB{err: dbErr}
	query := &failingStopQuery{err: queryErr}
	err := (&RuntimeWiring{DBManager: db, QueryService: query}).Close(context.Background())
	require.ErrorIs(t, err, queryErr)
	require.ErrorIs(t, err, dbErr)
	require.True(t, db.closed)
	require.True(t, query.stopped)
}

func TestRuntimeFactoryFailureClosesInitializedDatabase(t *testing.T) {
	t.Parallel()
	db := &failingCloseDB{}
	deps := defaultRuntimeWiringDeps()
	deps.loadConfig = func() (*database.Config, error) { return testConfig(), nil }
	deps.newDB = func(*database.Config) database.DatabaseManager { return db }
	deps.initDB = func(context.Context, database.DatabaseManager, int) error { return nil }
	deps.buildQuery = func(context.Context, database.DatabaseManager, *database.Config, ports.Logger, core.MetricsCollector) (QueryRuntimeService, ports.QueryService, error) {
		return nil, nil, errors.New("query initialization failed")
	}
	_, err := buildRuntimeWiringWithDeps(context.Background(), core.NewDefaultLogger(core.LogLevelInfo), core.NewDefaultMetricsCollector(), deps)
	require.Error(t, err)
	require.True(t, db.closed, "the factory owns resources until it returns a runtime")
}
