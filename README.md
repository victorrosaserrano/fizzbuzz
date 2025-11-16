# FizzBuzz API

A high-performance REST API that implements the FizzBuzz algorithm with customizable parameters and request statistics tracking.

## Features

- **Custom FizzBuzz Algorithm**: Configurable divisors, replacement strings, and sequence limits
- **Statistics Tracking**: Monitor most frequently requested parameter combinations
- **Input Validation**: Comprehensive parameter validation with detailed error messages
- **Rate Limiting**: Per-IP rate limiting with configurable thresholds
- **Structured Logging**: JSON-formatted logs with request correlation IDs
- **Health Monitoring**: Health check endpoint for operational monitoring
- **Graceful Shutdown**: Proper cleanup on shutdown signals

## Project Structure

```
fizzbuzz/
├── cmd/api/                    # Application entry point
├── internal/                   # Private packages
│   ├── data/                  # Business logic and data structures
│   └── validator/             # Input validation framework
├── bin/                       # Compiled binaries (build output)
├── migrations/                # Future database migration files
├── remote/                    # Deployment scripts and configurations
├── Makefile                   # Build automation
├── go.mod                     # Go module definition
├── go.sum                     # Dependency verification
├── README.md                  # Project documentation
└── .gitignore                 # Git ignore patterns
```

## Prerequisites

- Go 1.24 or later
- Make (for build automation)

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd fizzbuzz
```

### 2. Build the Application

```bash
make build
```

### 3. Run the Application

```bash
make run
```

The API will be available at `http://localhost:4000`.

## API Endpoints

### POST /v1/fizzbuzz

Generate FizzBuzz sequence with custom parameters.

**Request:**
```json
{
  "int1": 3,
  "int2": 5,
  "limit": 100,
  "str1": "fizz",
  "str2": "buzz"
}
```

**Response:**
```json
{
  "data": {
    "result": ["1", "2", "fizz", "4", "buzz", "fizz", "7", "8", "fizz", "buzz", ...]
  }
}
```

### GET /v1/statistics

Get the most frequently requested parameter combination.

**Response:**
```json
{
  "data": {
    "most_frequent_request": {
      "int1": 3,
      "int2": 5,
      "limit": 100,
      "str1": "fizz",
      "str2": "buzz"
    },
    "hits": 42
  }
}
```

### GET /v1/healthcheck

Health check endpoint.

**Response:**
```json
{
  "data": {
    "status": "available",
    "system_info": {
      "environment": "development",
      "version": "1.0.0"
    }
  }
}
```

## Development

### Available Make Commands

```bash
make help           # Show all available commands
make run            # Run the application
make build          # Build the application
make test           # Run tests
make audit          # Run quality control checks
```

### Development Workflow

1. **Code Quality**: Run `make audit` before committing changes
2. **Testing**: Run `make test` to execute the test suite
3. **Building**: Use `make build` to create production binaries

## Configuration

The application accepts the following command-line flags:

- `-port`: HTTP server port (default: 4000)
- `-limiter-rps`: Rate limiter requests per second (default: 2)
- `-limiter-burst`: Rate limiter burst size (default: 4)
- `-limiter-enabled`: Enable/disable rate limiting (default: true)

Example:
```bash
./bin/api -port=8080 -limiter-rps=10 -limiter-burst=20
```

## Architecture

The application follows the "Let's Go Further" architecture patterns:

- **Clean Architecture**: Clear separation between HTTP layer (`cmd/api`) and business logic (`internal`)
- **Dependency Injection**: Application dependencies injected at startup
- **Middleware Chain**: Logging, rate limiting, and recovery middleware
- **Custom Validation**: Type-safe input validation with detailed error messages
- **Structured Logging**: JSON logs with correlation IDs for request tracing

## Performance

- **Target**: 1000+ requests/second with sub-10ms response times
- **HTTP Router**: Zero-allocation routing via httprouter
- **Memory Management**: Pre-allocated slices and efficient data structures
- **Concurrency**: Thread-safe statistics tracking with RWMutex

## Contributing

1. Run `make audit` to ensure code quality
2. Add tests for new functionality
3. Update documentation as needed
4. Follow Go naming conventions and project patterns

## License

This project is part of a technical interview assignment.