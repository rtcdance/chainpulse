package database

import "errors"

// Error definitions for database operations
var (
	ErrMongoClientNotInitialized     = errors.New("MongoDB client not initialized")
	ErrPostgresDBNotInitialized      = errors.New("PostgreSQL database not initialized")
	ErrDatabaseManagerNotInitialized = errors.New("database manager not initialized")
	ErrDatabaseManagerAlreadyClosed  = errors.New("database manager already closed")
)
