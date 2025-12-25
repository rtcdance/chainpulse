# chainpulse

chainpulse is a comprehensive Web3 indexer and data service that monitors and indexes blockchain events such as NFT mints and token transfers. Built with Go, it provides real-time event processing, storage, and API access to blockchain data.

## Features

- **Real-time Event Listening**: Monitors blockchain events in real-time using Ethereum RPC
- **Event Indexing**: Stores NFT mint and token transfer events in PostgreSQL
- **Concurrent Processing**: Uses goroutines for efficient event processing
- **Caching Layer**: Implements Redis caching for hot queries
- **High-Performance JSON**: Optimized JSON serialization using go-json library (3x faster than standard library)
- **API Access**: Provides REST API for querying indexed data
- **Resume & Replay**: Supports breakpoint resume and event replay functionality
- **Enterprise Ready**: Includes logging, configuration management, and Docker support

## Architecture

For detailed architecture information, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

The system follows a microservice architecture with the following main components:

- `cmd/`: Main entry points for different services
- `services/`: Service-specific business logic
- `shared/`: Shared libraries and utilities used across services
- `proto/`: Protocol buffer definitions for gRPC services
- `test/`: Integration and end-to-end tests

## Prerequisites

- Go 1.25+
- PostgreSQL
- Redis
- Ethereum Node access (e.g., Infura, Alchemy)

## Getting Started

### Local Development

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd chainpulse
   ```

2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

3. Update the `.env` file with your configuration:
   - Set your Ethereum node URL
   - Configure PostgreSQL and Redis connections

4. Install dependencies:
   ```bash
   go mod tidy
   ```

5. Run the application:
   ```bash
   go run cmd/api/main.go
   ```

### Using Docker

1. Build and run with Docker Compose:
   ```bash
   cd build && docker-compose up -d
   ```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/events` - Get indexed events with filters
- `GET /api/v1/events/nft` - Get NFT transfer events
- `GET /api/v1/events/token` - Get token transfer events

### Query Parameters

- `type`: Event type (NFTTransfer, TokenTransfer)
- `contract`: Contract address
- `from_block`: Starting block number
- `to_block`: Ending block number
- `limit`: Number of results (default: 100)
- `offset`: Offset for pagination

## Configuration

The application uses environment variables for configuration:

- `ETHEREUM_NODE_URL`: Ethereum node endpoint
- `POSTGRESQL_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `PORT`: Server port (default: 8080)

## Development

The project follows Go best practices and enterprise-level code organization. Key design patterns include:

- Dependency injection
- Interface-based design
- Context-aware operations
- Error handling and logging
- Graceful shutdown

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.