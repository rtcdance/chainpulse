# ChainPulse Claude Rules

## Project Overview
ChainPulse is a Web3 blockchain event indexing system with enterprise-grade architecture.

## Development Guidelines

### Code Style
- Use Go 1.21+ features
- Follow Go standard conventions
- No unnecessary comments
- Table-driven tests preferred

### Architecture
- DDD layered architecture
- Plugin-based design
- Dependency inversion principle
- Interface-first approach

### Security
- Never expose secrets in code or logs
- Validate all inputs
- Use parameterized queries
- Implement proper authentication

### Testing
- Unit tests for all core logic
- Integration tests for critical paths
- Aim for >80% coverage
- Use mocks for external dependencies

## Project Structure

```
chainpulse/
├── cmd/           # Entry points
├── pkg/           # Public packages
│   ├── core/      # Domain types
│   ├── plugins/   # Implementations
│   └── services/  # Business logic
├── docs/          # Documentation
├── scripts/       # Utility scripts
└── test/          # Test suites
```

## Commands

```bash
make build          # Build all services
make test           # Run tests
make lint           # Run linter
make docker-up      # Start Docker services
```

## Constraints

### Must Do
- Handle all errors with context
- Use context for cancellation
- Write tests for new features
- Update documentation

### Must Not Do
- Commit secrets or credentials
- Use global variables
- Ignore errors
- Create deep nesting

## Web3 Specific

### Blockchain Data
- Always handle reorg scenarios
- Implement idempotency checks
- Use proper confirmation depth
- Support multiple chains

### Event Processing
- Validate event data
- Handle missing blocks
- Implement retry logic
- Track processing state
