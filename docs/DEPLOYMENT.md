# PDP Server Deployment Guide

## Prerequisites

### System Requirements
- **Operating System**: Linux (Ubuntu 20.04+ recommended), macOS, or Windows
- **Go Version**: 1.21 or higher
- **Memory**: Minimum 2GB RAM, 4GB+ recommended
- **Storage**: Minimum 10GB free space for data and blobs
- **Network**: Stable internet connection for blockchain interactions

### External Dependencies
- **Lotus Node**: For Filecoin network access (or use hosted endpoint)
- **Ethereum Client**: For Ethereum network access (or use hosted RPC)
- **Piri Wallet**: Existing wallet setup or new wallet creation

## Installation

### 1. Clone and Build

```bash
# Clone the repository
git clone <repository-url>
cd pdp-server

# Install dependencies
go mod download

# Build the binary
go build -o pdp-server cmd/server/main.go
```

### 2. Configuration Setup

Create a `config.yaml` file:

```yaml
server:
  host: "localhost"
  port: 8081

pdp:
  data_dir: "/opt/pdp-server/data"
  lotus_url: "wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1"
  eth_address: "0xYourEthereumAddress"
  key_file: "/opt/pdp-server/service.pem"

piri:
  database_path: "/opt/pdp-server/piri.db"
  wallet_path: "/opt/pdp-server/wallet"
```

### 3. Wallet Setup

#### Option A: Use Existing Piri Wallet
```bash
# Copy existing wallet files
cp -r ~/.storacha/wallet /opt/pdp-server/wallet
```

#### Option B: Create New Wallet
```bash
# Generate new private key
openssl ecparam -genkey -name secp256k1 -noout -out service.pem

# Set proper permissions
chmod 600 service.pem
```

### 4. Directory Structure Setup

```bash
# Create necessary directories
sudo mkdir -p /opt/pdp-server/{data,blobs,wallet}

# Set proper ownership (if running as non-root)
sudo chown -R $USER:$USER /opt/pdp-server

# Set proper permissions
chmod 755 /opt/pdp-server
chmod 755 /opt/pdp-server/{data,blobs,wallet}
```

## Configuration

### Environment Variables

The server supports configuration via environment variables:

```bash
export PDP_SERVER_HOST=localhost
export PDP_SERVER_PORT=8081
export PDP_DATA_DIR=/opt/pdp-server/data
export PDP_LOTUS_URL=wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1
export PDP_ETH_ADDRESS=0xYourEthereumAddress
export PDP_KEY_FILE=/opt/pdp-server/service.pem
```

### Configuration File Options

#### Server Configuration
```yaml
server:
  host: "0.0.0.0"           # Bind address (use 0.0.0.0 for all interfaces)
  port: 8081                # Server port
  read_timeout: "30s"       # Request read timeout
  write_timeout: "30s"      # Response write timeout
  idle_timeout: "120s"      # Connection idle timeout
```

#### PDP Configuration
```yaml
pdp:
  data_dir: "/opt/pdp-server/data"                          # Data directory
  blob_dir: "/opt/pdp-server/blobs"                         # Blob storage directory
  lotus_url: "wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1"  # Lotus endpoint
  eth_rpc_url: "https://eth-sepolia.public.blastapi.io"     # Ethereum RPC endpoint
  eth_address: "0xYourEthereumAddress"                      # Your Ethereum address
  key_file: "/opt/pdp-server/service.pem"                   # Private key file
```

#### Database Configuration
```yaml
database:
  type: "sqlite"                                    # Database type
  path: "/opt/pdp-server/data/pdp.db"              # Database file path
  max_open_conns: 25                               # Maximum open connections
  max_idle_conns: 5                                # Maximum idle connections
  conn_max_lifetime: "1h"                          # Connection maximum lifetime
```

#### Logging Configuration
```yaml
logging:
  level: "info"                     # Log level: debug, info, warn, error
  format: "json"                    # Log format: json, text
  output: "/var/log/pdp-server.log" # Log output file (or stdout)
```

## Running the Server

### Development Mode

```bash
# Run directly
./pdp-server

# Run with custom config
./pdp-server -config /path/to/config.yaml

# Run with debug logging
PDP_LOG_LEVEL=debug ./pdp-server
```

### Production Mode

#### Using Systemd (Recommended)

Create `/etc/systemd/system/pdp-server.service`:

```ini
[Unit]
Description=PDP Server
After=network.target

[Service]
Type=simple
User=pdp-server
Group=pdp-server
WorkingDirectory=/opt/pdp-server
ExecStart=/opt/pdp-server/pdp-server -config /opt/pdp-server/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pdp-server

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/pdp-server

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
# Create user and group
sudo useradd -r -s /bin/false pdp-server
sudo chown -R pdp-server:pdp-server /opt/pdp-server

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable pdp-server
sudo systemctl start pdp-server

# Check status
sudo systemctl status pdp-server
```

#### Using Docker

Create `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o pdp-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/pdp-server .
COPY --from=builder /app/config.yaml .

EXPOSE 8081
CMD ["./pdp-server"]
```

Build and run:

```bash
# Build image
docker build -t pdp-server:latest .

# Run container
docker run -d \
  --name pdp-server \
  -p 8081:8081 \
  -v /opt/pdp-server/data:/data \
  -v /opt/pdp-server/config.yaml:/config.yaml \
  pdp-server:latest
```

#### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  pdp-server:
    build: .
    ports:
      - "8081:8081"
    volumes:
      - ./data:/data
      - ./config.yaml:/config.yaml
      - ./service.pem:/service.pem
    environment:
      - PDP_DATA_DIR=/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Run with:

```bash
docker-compose up -d
```

## Reverse Proxy Setup

### Nginx Configuration

Create `/etc/nginx/sites-available/pdp-server`:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Increase timeouts for long-running operations
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Increase body size for file uploads
        client_max_body_size 100M;
    }
}
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/pdp-server /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### SSL/TLS with Let's Encrypt

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal is set up automatically
```

## Monitoring and Logging

### Health Checks

The server provides health check endpoints:

```bash
# Basic health check
curl http://localhost:8081/health

# Detailed status
curl http://localhost:8081/status
```

### Log Management

#### Logrotate Configuration

Create `/etc/logrotate.d/pdp-server`:

```
/var/log/pdp-server.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 pdp-server pdp-server
    postrotate
        systemctl reload pdp-server
    endscript
}
```

### Monitoring with Prometheus (Future)

The server will support Prometheus metrics at `/metrics` endpoint.

Example Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'pdp-server'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

## Security Considerations

### Firewall Configuration

```bash
# Allow SSH (if needed)
sudo ufw allow ssh

# Allow HTTP/HTTPS (if using reverse proxy)
sudo ufw allow 80
sudo ufw allow 443

# Allow PDP server port (if direct access needed)
sudo ufw allow 8081

# Enable firewall
sudo ufw enable
```

### File Permissions

```bash
# Secure configuration files
chmod 600 /opt/pdp-server/config.yaml
chmod 600 /opt/pdp-server/service.pem

# Secure data directory
chmod 750 /opt/pdp-server/data
chmod 750 /opt/pdp-server/blobs
```

### Network Security

- Use HTTPS in production (reverse proxy with SSL)
- Implement rate limiting at reverse proxy level
- Consider VPN access for administrative operations
- Regular security updates for the operating system

## Backup and Recovery

### Database Backup

```bash
#!/bin/bash
# backup-pdp.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/opt/backups/pdp-server"
DATA_DIR="/opt/pdp-server/data"

mkdir -p $BACKUP_DIR

# Stop service
sudo systemctl stop pdp-server

# Backup database and configuration
tar -czf $BACKUP_DIR/pdp-backup-$DATE.tar.gz \
    $DATA_DIR \
    /opt/pdp-server/config.yaml \
    /opt/pdp-server/service.pem

# Start service
sudo systemctl start pdp-server

# Keep only last 30 days of backups
find $BACKUP_DIR -name "pdp-backup-*.tar.gz" -mtime +30 -delete
```

### Recovery Procedure

```bash
# Stop service
sudo systemctl stop pdp-server

# Restore from backup
tar -xzf /opt/backups/pdp-server/pdp-backup-YYYYMMDD_HHMMSS.tar.gz -C /

# Start service
sudo systemctl start pdp-server

# Verify service
curl http://localhost:8081/health
```

## Performance Tuning

### Go Runtime Tuning

```bash
# Set Go runtime variables
export GOGC=100
export GOMAXPROCS=4
export GOMEMLIMIT=2GiB
```

### Database Optimization

For high-load scenarios, consider:
- Using PostgreSQL instead of SQLite
- Connection pooling optimization
- Database indexing optimization

### File System Optimization

```bash
# Mount data directory with appropriate options
/dev/sdb1 /opt/pdp-server/data ext4 defaults,noatime,nodiratime 0 2
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check service status
sudo systemctl status pdp-server

# Check logs
sudo journalctl -u pdp-server -f

# Check configuration
./pdp-server -config /opt/pdp-server/config.yaml -validate
```

#### Database Connection Issues

```bash
# Check database file permissions
ls -la /opt/pdp-server/data/

# Check disk space
df -h /opt/pdp-server/

# Test database connectivity
sqlite3 /opt/pdp-server/data/pdp.db ".tables"
```

#### Blockchain Connection Issues

```bash
# Test Lotus connection
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"Filecoin.Version","params":[],"id":1}' \
  wss://wss.calibration.node.glif.io/apigw/lotus/rpc/v1

# Check wallet file
openssl ec -in /opt/pdp-server/service.pem -text -noout
```

### Log Analysis

```bash
# Filter error logs
sudo journalctl -u pdp-server | grep ERROR

# Monitor real-time logs
sudo journalctl -u pdp-server -f

# Export logs for analysis
sudo journalctl -u pdp-server --since "1 hour ago" > /tmp/pdp-logs.txt
```

## Scaling Considerations

### Horizontal Scaling

For high availability and load distribution:

1. **Load Balancer**: Use nginx or HAProxy
2. **Shared Database**: Migrate to PostgreSQL
3. **Shared Storage**: Use network-attached storage
4. **Service Discovery**: Implement health check-based routing

### Vertical Scaling

- Increase server resources (CPU, RAM, storage)
- Optimize Go runtime parameters
- Tune database connection pools
- Implement caching strategies

## Migration Guide

### Upgrading Versions

```bash
# Stop service
sudo systemctl stop pdp-server

# Backup current installation
cp -r /opt/pdp-server /opt/pdp-server.backup.$(date +%Y%m%d)

# Deploy new version
cp new-pdp-server /opt/pdp-server/pdp-server

# Run database migrations (if needed)
./pdp-server -migrate

# Start service
sudo systemctl start pdp-server

# Verify upgrade
curl http://localhost:8081/health
```

### Configuration Changes

Always test configuration changes in a staging environment before applying to production.

```bash
# Validate configuration
./pdp-server -config config.yaml -validate

# Test with new configuration
./pdp-server -config config.yaml -dry-run
```
