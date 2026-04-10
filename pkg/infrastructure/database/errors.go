package database

import "fmt"

// Error definitions for database operations
var (
	ErrMongoClientNotInitialized     = fmt.Errorf("MongoDB client not initialized")
	ErrPostgresDBNotInitialized      = fmt.Errorf("PostgreSQL database not initialized")
	ErrDatabaseManagerNotInitialized = fmt.Errorf("database manager not initialized")
	ErrDatabaseManagerAlreadyClosed  = fmt.Errorf("database manager already closed")
)
