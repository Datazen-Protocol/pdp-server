# PDP Server

A standalone Proof of Data Possession (PDP) server that integrates with [Piri](https://github.com/storacha/piri) for Filecoin and Ethereum blockchain operations.

## 🚀 Quick Start

```bash
# Clone and build
git clone <repository-url>
cd pdp-server
go mod download
go build -o pdp-server cmd/server/main.go

# Configure (copy and edit config.yaml)
cp config.yaml.example config.yaml

# Run the server
./pdp-server

# Test all functionality
./test_complete.sh
```

Server will start at `http://localhost:8081`

## ✨ Features

- **🔗 Filecoin Integration**: CommP calculation with power-of-2 padding
- **📦 Proof Set Management**: Create and manage blockchain proof sets
- **🔄 Transaction Monitoring**: Automatic blockchain transaction tracking
- **💾 Isolated Database**: Independent SQLite database with GORM
- **🌐 RESTful API**: 19 HTTP endpoints for complete functionality
- **⚡ Automatic Padding**: Power-of-2 padding for Filecoin compatibility
- **🔒 Wallet Integration**: Ethereum wallet support via Piri
- **📁 Blob Storage**: File system-based piece storage

## 🏗️ Architecture

The PDP Server uses a dual-database architecture with clean separation:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PDP Server    │    │   Piri Library  │    │   Blockchain    │
│  (This Project) │    │ (External Dep)  │    │ (Filecoin/ETH)  │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • Local SQLite  │◄──►│ • PDP Service   │◄──►│ • Smart Contract│
│ • Piece Storage │    │ • Piri Database │    │ • Transactions  │
│ • HTTP API      │    │ • Wallet Mgmt   │    │ • Events        │
│ • Tx Watcher    │    │ • Blockchain    │    │ • Confirmations │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 📡 API Endpoints

### Health & Status
- `GET /health` - Health check
- `GET /status` - Server status and information

### File Management
- `POST /upload` - Direct file upload
- `GET /files` - List uploaded files

### Piece Management
- `POST /pieces` - Prepare piece for upload
- `PUT /pieces/:pieceID` - Upload piece data
- `GET /pieces/:pieceID` - Get piece information
- `POST /pieces/:pieceID/proofset/:proofSetID` - Add piece to proof set

### Proof Set Management
- `POST /proofsets` - Create new proof set
- `GET /proofsets` - List all proof sets
- `GET /proofsets/:id` - Get proof set details
- `POST /proofsets/:id/roots` - Add roots to proof set
- `GET /proofsets/:id/roots` - Get proof set roots
- `GET /proofsets/:id/status` - Get detailed proof set status

### Proving Operations
- `POST /proofsets/:id/prove` - Trigger proving
- `GET /proofsets/:id/prove/status` - Get proving status

### Transaction Monitoring
- `GET /pieces/:pieceID/transaction/status` - Get transaction status
- `POST /pieces/:pieceID/transaction/monitor` - Monitor transaction

**Total: 19 endpoints** | 📖 [Complete API Reference](docs/API.md)

## ⚙️ Configuration

Create a `config.yaml` file:

```yaml
server:
  host: "localhost"
  port: 8081

pdp:
  data_dir: "/opt/pdp-server/data"
  lotus_url: "wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1"
  eth_address: "0x2F3DAD0e140B7c93a13DC54329725704063b9d4A"
  key_file: "/opt/pdp-server/service.pem"

piri:
  database_path: "/opt/pdp-server/piri.db"
  wallet_path: "/opt/pdp-server/wallet"
```

🔧 [Complete Configuration Guide](docs/DEPLOYMENT.md#configuration)

## 🧪 Testing

Run the comprehensive end-to-end test suite:

```bash
# Full test suite (recommended)
./test_complete.sh

# Individual API tests
curl http://localhost:8081/health
curl -X POST http://localhost:8081/proofsets
curl -X POST -F "file=@test.txt" http://localhost:8081/upload
```

The test suite validates:
- ✅ Server health and status endpoints
- ✅ File uploads with power-of-2 padding
- ✅ Piece preparation and upload workflow
- ✅ Proof set creation on blockchain
- ✅ Root addition to proof sets
- ✅ Transaction monitoring and confirmations
- ✅ Database persistence and blob storage
- ✅ Piri integration and wallet operations

## 📋 Requirements

- **Go**: 1.21 or later
- **System**: Linux, macOS, or Windows
- **Memory**: 2GB+ RAM recommended
- **Storage**: 10GB+ free space
- **Network**: Internet connection for blockchain access
- **Dependencies**: All Go modules auto-installed via `go mod download`

### External Services (Optional)
- **Lotus Node**: Use hosted endpoint or run your own
- **Ethereum RPC**: Use public endpoints or run your own
- **Wallet**: Existing Piri wallet or create new one

## 🔧 Development

### Local Development Setup

```bash
# Install dependencies
go mod download

# Build the server
go build -o pdp-server cmd/server/main.go

# Run in development mode
go run cmd/server/main.go

# Run with debug logging
PDP_LOG_LEVEL=debug ./pdp-server
```

### Example API Usage

```bash
# Health check
curl http://localhost:8081/health

# Create a proof set
curl -X POST http://localhost:8081/proofsets

# Upload a file
curl -X POST -F "file=@example.txt" http://localhost:8081/upload

# Prepare and upload a piece
curl -X POST -H "Content-Type: application/json" \
  -d '{"check":{"name":"sha2-256","hash":"abc123","size":1024}}' \
  http://localhost:8081/pieces

curl -X PUT -H "Content-Type: application/octet-stream" \
  --data-binary @piece-data.bin \
  http://localhost:8081/pieces/piece-uuid
```

## 📚 Documentation

### 📖 Complete Documentation
- **[📋 API Reference](docs/API.md)** - Complete REST API documentation
- **[🏗️ Architecture Guide](docs/ARCHITECTURE.md)** - System design and components  
- **[🚀 Deployment Guide](docs/DEPLOYMENT.md)** - Production setup and configuration
- **[🤝 Contributing Guide](docs/CONTRIBUTING.md)** - Development workflow and standards

### 🔗 Quick Links
- [📡 All API Endpoints](docs/API.md) - 19 endpoints with examples
- [⚙️ Configuration Options](docs/DEPLOYMENT.md#configuration) - Environment setup
- [🐳 Docker Setup](docs/DEPLOYMENT.md#using-docker) - Container deployment
- [🔒 Security Guide](docs/DEPLOYMENT.md#security-considerations) - Security best practices

## 🔒 Security

- **Private Keys**: Stored in `service.pem` - keep secure and never commit!
- **Database Isolation**: Independent SQLite database with proper permissions
- **Transaction Signing**: All blockchain transactions properly signed via Piri
- **Input Validation**: Comprehensive validation on all API endpoints
- **File Permissions**: Proper filesystem permissions for data directories

## 🤝 Integration

This server integrates with [Piri](https://github.com/storacha/piri) as an external Go package:

- **Clean Separation**: PDP Server maintains its own database and API
- **Service Adapter**: Clean interface between PDP Server and Piri services
- **Transaction Sync**: Background watcher synchronizes blockchain state
- **Wallet Reuse**: Leverages existing Piri wallet infrastructure

## 📄 License

MIT License - see LICENSE file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/your-org/pdp-server/issues)
- **Documentation**: [docs/](docs/) directory
- **Contributing**: [CONTRIBUTING.md](docs/CONTRIBUTING.md)

---

🎉 **Production Ready** | 🔗 **Filecoin Compatible** | ⚡ **High Performance**