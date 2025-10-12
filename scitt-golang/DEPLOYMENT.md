# SCITT Transparency Service - Deployment Guide

**Version**: 1.0.0
**Status**: Production Ready
**Last Updated**: 2025-10-12

---

## Pre-Deployment Checklist

### ✅ Prerequisites

- [ ] Go 1.24 or later installed
- [ ] SQLite support (CGO enabled)
- [ ] Sufficient disk space (minimum 1GB)
- [ ] Network connectivity for HTTP server
- [ ] TLS certificates (if using HTTPS)

### ✅ System Requirements

**Minimum**:
- CPU: 1 core
- RAM: 512MB
- Disk: 1GB
- OS: Linux, macOS, or Windows

**Recommended**:
- CPU: 2+ cores
- RAM: 2GB
- Disk: 10GB (with room for growth)
- OS: Linux (Ubuntu 22.04+ or RHEL 8+)

---

## Quick Start Deployment

### 1. Build the Binary

```bash
# Clone repository
git clone <repository-url>
cd transparency-service/scitt-golang

# Run tests
go test ./...

# Build binary
go build -o scitt ./cmd/scitt

# Verify build
./scitt --version
```

### 2. Initialize Service

```bash
# Create service directory
mkdir -p /opt/scitt
cd /opt/scitt

# Initialize transparency service
./scitt init --origin https://transparency.example.com

# Verify initialization
ls -la
# Should see: service-key.pem, service-key.jwk, scitt.db, storage/, scitt.yaml
```

### 3. Configure Service

Edit `scitt.yaml`:

```yaml
origin: https://transparency.example.com

database:
  path: /opt/scitt/scitt.db
  enable_wal: true

storage:
  type: local
  path: /opt/scitt/storage

keys:
  private: /opt/scitt/service-key.pem
  public: /opt/scitt/service-key.jwk

server:
  host: 0.0.0.0
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "https://your-domain.com"
```

### 4. Start Service

```bash
# Run in foreground (testing)
./scitt serve --config scitt.yaml

# Or run as systemd service (production)
sudo systemctl start scitt
```

---

## Production Deployment

### Option 1: Systemd Service (Linux)

#### Create Service File

`/etc/systemd/system/scitt.service`:

```ini
[Unit]
Description=SCITT Transparency Service
After=network.target

[Service]
Type=simple
User=scitt
Group=scitt
WorkingDirectory=/opt/scitt
ExecStart=/opt/scitt/scitt serve --config /opt/scitt/scitt.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/scitt

[Install]
WantedBy=multi-user.target
```

#### Install and Start

```bash
# Create user
sudo useradd -r -s /bin/false scitt

# Set permissions
sudo chown -R scitt:scitt /opt/scitt

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable scitt
sudo systemctl start scitt

# Check status
sudo systemctl status scitt

# View logs
sudo journalctl -u scitt -f
```

---

### Option 2: Docker Deployment

#### Dockerfile

```dockerfile
# Build stage
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o scitt ./cmd/scitt

# Runtime stage
FROM debian:bookworm-slim

# Install SQLite and CA certificates
RUN apt-get update && \
    apt-get install -y ca-certificates sqlite3 && \
    rm -rf /var/lib/apt/lists/*

# Create service user
RUN useradd -r -s /bin/false scitt

# Copy binary
COPY --from=builder /app/scitt /usr/local/bin/scitt

# Create directories
RUN mkdir -p /var/lib/scitt/storage && \
    chown -R scitt:scitt /var/lib/scitt

# Switch to service user
USER scitt
WORKDIR /var/lib/scitt

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/usr/local/bin/scitt", "--version"]

ENTRYPOINT ["/usr/local/bin/scitt"]
CMD ["serve", "--config", "/var/lib/scitt/scitt.yaml"]
```

#### Build and Run

```bash
# Build image
docker build -t scitt-transparency-service:1.0.0 .

# Run container
docker run -d \
  --name scitt \
  -p 8080:8080 \
  -v /opt/scitt:/var/lib/scitt \
  --restart unless-stopped \
  scitt-transparency-service:1.0.0

# Check logs
docker logs -f scitt

# Check health
docker inspect --format='{{.State.Health.Status}}' scitt
```

#### Docker Compose

`docker-compose.yml`:

```yaml
version: '3.8'

services:
  scitt:
    build: .
    container_name: scitt
    ports:
      - "8080:8080"
    volumes:
      - scitt-data:/var/lib/scitt
    environment:
      - SCITT_ORIGIN=https://transparency.example.com
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "/usr/local/bin/scitt", "--version"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  scitt-data:
    driver: local
```

---

### Option 3: Kubernetes Deployment

#### ConfigMap

`configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scitt-config
data:
  scitt.yaml: |
    origin: https://transparency.example.com
    database:
      path: /data/scitt.db
      enable_wal: true
    storage:
      type: local
      path: /data/storage
    keys:
      private: /keys/service-key.pem
      public: /keys/service-key.jwk
    server:
      host: 0.0.0.0
      port: 8080
      cors:
        enabled: true
        allowed_origins: ["*"]
```

#### Deployment

`deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scitt
spec:
  replicas: 1
  selector:
    matchLabels:
      app: scitt
  template:
    metadata:
      labels:
        app: scitt
    spec:
      containers:
      - name: scitt
        image: scitt-transparency-service:1.0.0
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /var/lib/scitt/scitt.yaml
          subPath: scitt.yaml
        - name: data
          mountPath: /data
        - name: keys
          mountPath: /keys
          readOnly: true
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: scitt-config
      - name: data
        persistentVolumeClaim:
          claimName: scitt-data
      - name: keys
        secret:
          secretName: scitt-keys
```

#### Service

`service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: scitt
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
  selector:
    app: scitt
```

---

## Configuration

### Environment-Specific Configs

#### Development

```yaml
origin: http://localhost:8080

database:
  path: ./dev.db
  enable_wal: false

storage:
  type: memory

server:
  host: 127.0.0.1
  port: 8080
  cors:
    enabled: true
    allowed_origins: ["*"]
```

#### Staging

```yaml
origin: https://staging.transparency.example.com

database:
  path: /var/lib/scitt/staging.db
  enable_wal: true

storage:
  type: local
  path: /var/lib/scitt/storage

server:
  host: 0.0.0.0
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "https://staging.example.com"
```

#### Production

```yaml
origin: https://transparency.example.com

database:
  path: /var/lib/scitt/prod.db
  enable_wal: true

storage:
  type: local  # or minio when available
  path: /var/lib/scitt/storage

server:
  host: 0.0.0.0
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "https://example.com"
      - "https://www.example.com"
```

---

## Security

### Key Management

```bash
# Generate keys during initialization
./scitt init --origin https://transparency.example.com

# Secure private key
chmod 600 service-key.pem
chown scitt:scitt service-key.pem

# Backup keys securely
tar -czf scitt-keys-backup.tar.gz service-key.pem service-key.jwk
gpg --encrypt --recipient admin@example.com scitt-keys-backup.tar.gz

# Store backup in secure location
mv scitt-keys-backup.tar.gz.gpg /secure/backup/location/
```

### TLS/HTTPS

Use a reverse proxy (nginx, Apache, Caddy) for TLS termination:

#### Nginx Configuration

```nginx
upstream scitt_backend {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name transparency.example.com;

    ssl_certificate /etc/ssl/certs/transparency.example.com.crt;
    ssl_certificate_key /etc/ssl/private/transparency.example.com.key;

    location / {
        proxy_pass http://scitt_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://scitt_backend/health;
        access_log off;
    }
}
```

### Firewall Rules

```bash
# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Block direct access to application port
sudo ufw deny 8080/tcp

# Enable firewall
sudo ufw enable
```

---

## Monitoring

### Health Checks

```bash
# Basic health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","origin":"https://transparency.example.com"}

# Detailed health check script
#!/bin/bash
HEALTH_URL="http://localhost:8080/health"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ "$RESPONSE" -eq 200 ]; then
    echo "Service is healthy"
    exit 0
else
    echo "Service is unhealthy (HTTP $RESPONSE)"
    exit 1
fi
```

### Logging

```bash
# Systemd logs
sudo journalctl -u scitt -f

# Docker logs
docker logs -f scitt

# Log rotation (systemd automatically rotates journal logs)
# For custom logging, configure logrotate:
/var/log/scitt/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 scitt scitt
    sharedscripts
    postrotate
        systemctl reload scitt > /dev/null 2>&1 || true
    endscript
}
```

### Metrics

Monitor these key metrics:

1. **Request Rate**: Requests per second to `/entries`
2. **Response Time**: Average response time for endpoints
3. **Tree Size**: Current transparency log size
4. **Database Size**: SQLite file size
5. **Storage Size**: Storage directory size
6. **Error Rate**: 4xx and 5xx responses

Example monitoring script:

```bash
#!/bin/bash
# Basic metrics collection

# Tree size
TREE_SIZE=$(curl -s http://localhost:8080/checkpoint | head -2 | tail -1)

# Database size
DB_SIZE=$(du -h /opt/scitt/scitt.db | cut -f1)

# Storage size
STORAGE_SIZE=$(du -sh /opt/scitt/storage | cut -f1)

echo "Tree Size: $TREE_SIZE"
echo "Database Size: $DB_SIZE"
echo "Storage Size: $STORAGE_SIZE"
```

---

## Backup and Recovery

### Backup Strategy

```bash
#!/bin/bash
# backup.sh - Daily backup script

BACKUP_DIR="/backup/scitt/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# Stop service for consistent backup
systemctl stop scitt

# Backup database
cp /opt/scitt/scitt.db "$BACKUP_DIR/"
cp /opt/scitt/scitt.db-wal "$BACKUP_DIR/" 2>/dev/null || true
cp /opt/scitt/scitt.db-shm "$BACKUP_DIR/" 2>/dev/null || true

# Backup storage
tar -czf "$BACKUP_DIR/storage.tar.gz" /opt/scitt/storage/

# Backup keys
cp /opt/scitt/service-key.pem "$BACKUP_DIR/"
cp /opt/scitt/service-key.jwk "$BACKUP_DIR/"

# Backup config
cp /opt/scitt/scitt.yaml "$BACKUP_DIR/"

# Start service
systemctl start scitt

# Verify backup
if [ -f "$BACKUP_DIR/scitt.db" ]; then
    echo "Backup completed successfully: $BACKUP_DIR"
else
    echo "Backup failed!"
    exit 1
fi

# Cleanup old backups (keep 30 days)
find /backup/scitt/ -type d -mtime +30 -exec rm -rf {} +
```

### Recovery

```bash
#!/bin/bash
# restore.sh - Restore from backup

BACKUP_DATE="20251012"  # Specify backup date
BACKUP_DIR="/backup/scitt/$BACKUP_DATE"

# Stop service
systemctl stop scitt

# Restore database
cp "$BACKUP_DIR/scitt.db" /opt/scitt/

# Restore storage
rm -rf /opt/scitt/storage
tar -xzf "$BACKUP_DIR/storage.tar.gz" -C /

# Restore keys
cp "$BACKUP_DIR/service-key.pem" /opt/scitt/
cp "$BACKUP_DIR/service-key.jwk" /opt/scitt/

# Restore config
cp "$BACKUP_DIR/scitt.yaml" /opt/scitt/

# Set permissions
chown -R scitt:scitt /opt/scitt
chmod 600 /opt/scitt/service-key.pem

# Start service
systemctl start scitt

echo "Restore completed from $BACKUP_DIR"
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check service status
systemctl status scitt

# Check logs
journalctl -u scitt -n 100

# Common issues:
# 1. Port already in use
sudo lsof -i :8080

# 2. Database permissions
ls -la /opt/scitt/scitt.db
chmod 644 /opt/scitt/scitt.db

# 3. Missing keys
ls -la /opt/scitt/service-key.*
```

### Database Issues

```bash
# Check database integrity
sqlite3 /opt/scitt/scitt.db "PRAGMA integrity_check;"

# Verify WAL mode
sqlite3 /opt/scitt/scitt.db "PRAGMA journal_mode;"

# Checkpoint WAL
sqlite3 /opt/scitt/scitt.db "PRAGMA wal_checkpoint(FULL);"

# Rebuild database (last resort)
systemctl stop scitt
mv /opt/scitt/scitt.db /opt/scitt/scitt.db.backup
./scitt init --origin https://transparency.example.com
systemctl start scitt
```

### Performance Issues

```bash
# Check database size
du -h /opt/scitt/scitt.db

# Check storage size
du -sh /opt/scitt/storage

# Monitor resource usage
top -p $(pgrep scitt)

# Check disk I/O
iostat -x 1 10

# Optimize database
sqlite3 /opt/scitt/scitt.db "VACUUM;"
sqlite3 /opt/scitt/scitt.db "ANALYZE;"
```

---

## Upgrades

### Zero-Downtime Upgrade

```bash
# 1. Backup current version
./backup.sh

# 2. Build new version
go build -o scitt-new ./cmd/scitt

# 3. Test new version
./scitt-new --version

# 4. Stop old service
systemctl stop scitt

# 5. Replace binary
mv /opt/scitt/scitt /opt/scitt/scitt.old
mv scitt-new /opt/scitt/scitt

# 6. Start new service
systemctl start scitt

# 7. Verify
curl http://localhost:8080/health

# 8. If successful, remove old binary
rm /opt/scitt/scitt.old
```

---

## Testing in Production

```bash
# Register a test statement
echo '{"test": true}' > test-payload.json

scitt statement sign \
  --input test-payload.json \
  --key issuer-key.pem \
  --output test-statement.cbor \
  --issuer "https://test.example.com" \
  --subject "deployment-test"

# Register via API
curl -X POST https://transparency.example.com/entries \
  -H "Content-Type: application/cose" \
  --data-binary @test-statement.cbor

# Verify checkpoint
curl https://transparency.example.com/checkpoint

# Clean up
rm test-payload.json test-statement.cbor
```

---

## Post-Deployment Checklist

- [ ] Service is running (`systemctl status scitt`)
- [ ] Health endpoint returns 200 (`curl /health`)
- [ ] Can register statements (`POST /entries`)
- [ ] Can retrieve receipts (`GET /entries/{id}`)
- [ ] Checkpoint updates correctly (`GET /checkpoint`)
- [ ] Logs are being written
- [ ] Backups are configured
- [ ] Monitoring is set up
- [ ] TLS is configured (if applicable)
- [ ] Firewall rules are in place
- [ ] Documentation is accessible

---

## Support

For issues and questions:
- Check logs: `journalctl -u scitt -f`
- Review documentation: `README.md`, `TESTING-GUIDE.md`
- Run diagnostics: `scitt --version`, `scitt help`

---

*Last Updated: 2025-10-12*
*Version: 1.0.0*
