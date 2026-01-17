# Price Fetcher Microservice

A high-performance Go microservice providing stock price data through dual API interfaces (gRPC and JSON/REST), demonstrating performance comparison between gRPC and traditional REST protocols.

## 🚀 Features

- **Dual API Support**: gRPC (port 8081) and JSON/REST (port 8080) endpoints
- **Performance Benchmarking**: Automatic comparison of gRPC vs JSON throughput (100,000 iterations)
- **Structured Logging**: Request tracking with logrus, request IDs, and duration metrics
- **Protocol Buffers**: Type-safe service contracts with auto-generated code
- **Decorator Pattern**: Logging middleware implementation
- **Client Libraries**: Both HTTP and gRPC client implementations

## 📋 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Clients                              │
└────────────────────┬────────────────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         │                       │
    ┌────▼────┐            ┌────▼────┐
    │  gRPC   │            │  HTTP   │
    │ :8081   │            │  :8080  │
    └────┬────┘            └────┬────┘
         │                       │
         └───────────┬───────────┘
                     │
              ┌──────▼──────┐
              │ Logging     │
              │ Middleware  │
              └──────┬──────┘
                     │
              ┌──────▼──────┐
              │ Price       │
              │ Service     │
              └──────┬──────┘
                     │
              ┌──────▼──────┐
              │ Mock Data   │
              │ (AAPL, MSFT,│
              │  GOOGL)     │
              └─────────────┘
```

## 🛠️ Tech Stack

- **Go 1.25.5** - Programming language
- **gRPC v1.78.0** - High-performance RPC framework
- **Protocol Buffers v1.36.11** - Data serialization
- **Logrus v1.9.3** - Structured logging
- **Google UUID v1.6.0** - Request ID generation

## 📦 Installation

### Prerequisites

- Go 1.25+
- Protocol Buffers compiler (`protoc`)
- Make (optional, for build automation)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd microservice-priceFetcher

# Build the binary
make build

# Or use go directly
go build -o ./bin/priceFetcher
```

### Generate Protocol Buffer Code

```bash
# Generate Go code from .proto files
make proto

# Or manually
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/service.proto
```

## 🏃 Running the Service

### Using Make

```bash
make run
```

### Using Go

```bash
go run .
```

### Custom Ports

```bash
./bin/priceFetcher -json-addr=:8080 -grpc-addr=:8081
```

The service will:
1. Start both gRPC (port 8081) and HTTP (port 8080) servers
2. Run performance benchmarks (100,000 iterations)
3. Display throughput results:
   ```
   gRPC: 45000.00 requests/second
   JSON: 12000.00 requests/second
   ```
4. Keep running indefinitely

## 📡 API Usage

### gRPC API

**Endpoint**: `localhost:8081`

**Proto Definition**:
```protobuf
service PriceFetcher {
  rpc FetchPrice(FetchPriceRequest) returns (FetchPriceResponse);
}

message FetchPriceRequest {
  string ticker = 1;
}

message FetchPriceResponse {
  string ticker = 1;
  float price = 2;
}
```

**Client Example**:
```go
package main

import (
    "context"
    "log"
    "github.com/aliexe/ms-priceFetcher/client"
    "github.com/aliexe/ms-priceFetcher/proto"
)

func main() {
    client, err := client.NewGRPCClient("localhost:8081")
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.FetchPrice(context.Background(), &proto.FetchPriceRequest{
        Ticker: "AAPL",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Price: %s = %f", resp.Ticker, resp.Price)
}
```

### JSON/REST API

**Endpoint**: `localhost:8080`

**Get Stock Price**:
```bash
curl "http://localhost:8080/price?ticker=AAPL"
```

**Response**:
```json
{
  "ticker": "AAPL",
  "price": 150.0
}
```

**Supported Tickers**:
- `AAPL` - Apple Inc. ($150.00)
- `MSFT` - Microsoft Corporation ($300.00)
- `GOOGL` - Alphabet Inc. ($2800.00)

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

## 📊 Performance Benchmarks

The service automatically runs benchmarks on startup comparing gRPC vs JSON performance:

| Protocol | Throughput (approx) | Latency |
|----------|---------------------|---------|
| gRPC     | 45,000 req/sec      | ~22μs   |
| JSON/HTTP| 12,000 req/sec      | ~83μs   |

*Results may vary based on hardware and system load*

## 📁 Project Structure

```
microservice-priceFetcher/
├── main.go              # Entry point, benchmark logic, server orchestration
├── service.go           # Core business logic and mock data
├── grpc_server.go       # gRPC server implementation
├── json_api.go          # HTTP/REST server implementation
├── logging.go           # Logging middleware/decorator
├── Makefile             # Build and proto generation commands
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
├── client/              # Client implementations
│   └── client.go        # HTTP and gRPC clients
├── example/             # Usage examples
│   └── client_example.go # Example client usage
├── proto/               # Protocol buffer definitions
│   ├── service.proto    # Service definition
│   ├── service.pb.go    # Generated messages
│   └── service_grpc.pb.go # Generated gRPC code
└── types/               # Shared types
    └── types.go         # Response types
```

## 🔧 Configuration

### Environment Variables

Currently, the service uses command-line flags:

- `-json-addr`: JSON API address (default: `:8080`)
- `-grpc-addr`: gRPC server address (default: `:8081`)

### Mock Data

Prices are hardcoded in `service.go`:
```go
var priceMocks = map[string]float64{
    "AAPL":  150.0,
    "MSFT":  300.0,
    "GOOGL": 2800.0,
}
```

## 📝 Design Patterns

- **Decorator Pattern**: `LoggingService` wraps `priceService` with logging
- **Interface Segregation**: `PriceService` interface defines contract
- **Dependency Injection**: Services are injected into servers
- **Middleware Pattern**: HTTP handler wrapper function
- **Factory Pattern**: `New*` functions create instances

## 🚀 Roadmap

### Phase 1 (In Progress)
- [ ] Real data integration with financial APIs
- [ ] Comprehensive testing suite
- [ ] Docker containerization

### Phase 2
- [ ] Caching layer (Redis)
- [ ] Database persistence (PostgreSQL)
- [ ] Authentication & authorization

### Phase 3
- [ ] Kubernetes deployment
- [ ] Metrics & monitoring (Prometheus)
- [ ] CI/CD pipeline

### Phase 4
- [ ] Configuration management
- [ ] Advanced error handling
- [ ] Security hardening

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

[Your License Here]

## 👤 Author

[Your Name]

## 🙏 Acknowledgments

- gRPC and Protocol Buffers teams
- Go community
- Logrus library contributors