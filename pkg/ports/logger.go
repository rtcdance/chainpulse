package ports

// LeveledLogger provides basic leveled logging.
type LeveledLogger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Fatal(msg string, fields ...any)
}

// Logger provides structured logging with correlation ID support.
type Logger interface {
	LeveledLogger
	WithCorrelationID(id string) Logger
}
