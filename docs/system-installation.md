# Go-Doc-Go System Installation Guide

Complete guide for installing and deploying the full Go-Doc-Go document processing system including backend API, frontend UI, workers, and supporting infrastructure.

## System Architecture Overview

Go-Doc-Go is a distributed document processing system with these components:

- **Backend API Server**: Flask-based REST API with search, configuration, and pipeline management
- **Frontend UI**: React-based web interface for system management and document search
- **Document Workers**: Scalable workers for processing documents, embeddings, and relationships
- **Database Layer**: PostgreSQL for storage, Redis for caching and job queues
- **Monitoring Stack**: Prometheus, Grafana, and Loki for observability
- **Queue System**: Distributed work queue with leader election for coordinated processing

## Installation Options

### 1. Quick Start - Docker Compose (Recommended)

The fastest way to get the entire system running:

```bash
# Clone the repository
git clone https://github.com/kenstott/go-doc-go.git
cd go-doc-go

# Start with simple configuration
docker-compose -f docker-compose.simple.yml up -d

# Or start with full production stack
docker-compose -f docker-compose.prod.yml up -d
```

**Access URLs:**
- Frontend UI: http://localhost:80
- Backend API: http://localhost:8000
- API Documentation: http://localhost:8000/docs
- Grafana (if monitoring enabled): http://localhost:3001

### 2. Development Setup

For development and customization:

```bash
# Prerequisites
- Docker and Docker Compose
- Node.js 18+ (for frontend development)
- Python 3.9+ (for backend development)

# Development environment
docker-compose -f docker-compose.dev.yml up -d

# Frontend development server
cd frontend
npm install
npm run dev

# Backend development server  
pip install -e ".[dev,all]"
python -m go_doc_go.server
```

### 3. Manual Installation

For custom deployments and production environments.

---

## Docker Compose Deployments

### Simple Stack

**File**: `docker-compose.simple.yml`

Minimal setup with SQLite database:

```yaml
# Services included:
- Backend API (with SQLite)
- Frontend UI (served via backend)
- Basic monitoring
```

**Usage:**
```bash
# Start services
docker-compose -f docker-compose.simple.yml up -d

# View logs
docker-compose -f docker-compose.simple.yml logs -f

# Stop services
docker-compose -f docker-compose.simple.yml down
```

### Development Stack

**File**: `docker-compose.dev.yml`

Development environment with hot reloading:

```yaml
# Services included:
- Backend API (development mode)
- Frontend UI (development server)
- PostgreSQL database
- Redis cache
- Volume mounts for live code changes
```

**Setup:**
```bash
# Create environment file
cp .env.example .env

# Edit configuration
vim .env

# Start development environment
docker-compose -f docker-compose.dev.yml up -d

# Attach to services for debugging
docker-compose -f docker-compose.dev.yml exec backend bash
docker-compose -f docker-compose.dev.yml exec frontend bash
```

### Production Stack

**File**: `docker-compose.prod.yml`

Full production deployment with scaling and monitoring:

```yaml
# Services included:
- Backend API (production optimized)
- Frontend UI (Nginx-served static build)
- Scalable Worker processes (3 replicas by default)
- PostgreSQL database with persistence
- Redis for caching and job queues
- Prometheus metrics collection
- Grafana dashboards
- Loki log aggregation
- Automated backup service
```

**Production Setup:**

1. **Create secrets directory:**
```bash
mkdir -p secrets
echo "your_postgres_password" > secrets/postgres_password.txt
echo "your_openai_api_key" > secrets/openai_api_key.txt  
echo "your_anthropic_api_key" > secrets/anthropic_api_key.txt
echo "your_grafana_password" > secrets/grafana_password.txt
```

2. **Configure environment:**
```bash
# Create production environment file
cat > .env.prod << EOF
TAG=latest
DATABASE_URL=postgresql://go_doc_go:changeme@postgres:5432/go_doc_go
REDIS_URL=redis://redis:6379
LOG_LEVEL=INFO
CORS_ORIGINS=https://yourdomain.com
WORKER_REPLICAS=3
WORKER_MEMORY_LIMIT=4G
WORKER_CPU_LIMIT=2.0
BACKUP_SCHEDULE=0 2 * * *
NGINX_HOST=yourdomain.com
EOF
```

3. **Deploy production stack:**
```bash
# Start core services
docker-compose -f docker-compose.prod.yml up -d backend frontend postgres redis worker

# Start monitoring (optional)
docker-compose -f docker-compose.prod.yml --profile monitoring up -d

# Start backup service (optional)  
docker-compose -f docker-compose.prod.yml --profile backup up -d

# Check service health
docker-compose -f docker-compose.prod.yml ps
```

4. **Scaling workers:**
```bash
# Scale workers based on load
docker-compose -f docker-compose.prod.yml up -d --scale worker=5

# Or set in environment
export WORKER_REPLICAS=5
docker-compose -f docker-compose.prod.yml up -d
```

---

## Manual Installation

### Prerequisites

**System Requirements:**
- **OS**: Linux (Ubuntu 20.04+, RHEL 8+, CentOS 8+), macOS 11+, Windows 10+ (WSL2)
- **Memory**: 8GB RAM minimum (16GB+ recommended for production)
- **Storage**: 50GB+ available space
- **CPU**: 4+ cores recommended

**Software Dependencies:**
- Python 3.9+
- Node.js 18+
- PostgreSQL 13+ or SQLite 3.35+
- Redis 6+ (optional, for caching and job queues)
- Nginx (optional, for production frontend serving)

### Backend Installation

1. **Install Go-Doc-Go backend:**
```bash
# Install with all features
pip install "go-doc-go[all]"

# Or install with specific features for production
pip install "go-doc-go[db-postgresql,fastembed,cloud-aws]"
```

2. **Database Setup:**

**PostgreSQL** (Recommended for production):
```bash
# Install PostgreSQL
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql
postgres=# CREATE DATABASE go_doc_go;
postgres=# CREATE USER go_doc_go WITH ENCRYPTED PASSWORD 'your_password';
postgres=# GRANT ALL PRIVILEGES ON DATABASE go_doc_go TO go_doc_go;
postgres=# \q

# Initialize database schema
export DATABASE_URL="postgresql://go_doc_go:your_password@localhost/go_doc_go"
python -c "from go_doc_go import Config; Config().initialize_database()"
```

**SQLite** (Development/Testing):
```bash
# SQLite setup is automatic
export GO_DOC_GO_CONFIG_PATH=config.yaml
python -c "from go_doc_go import Config; Config('config.yaml').initialize_database()"
```

3. **Configuration:**
```bash
# Create configuration file
cat > config.yaml << EOF
storage:
  backend: postgresql  # or sqlite
  path: postgresql://go_doc_go:password@localhost/go_doc_go  # or ./go_doc_go.db

embedding:
  enabled: true
  model: sentence-transformers/all-MiniLM-L6-v2
  dimensions: 384

relationship_detection:
  enabled: true
  similarity_threshold: 0.7
  max_relationships_per_element: 5

content_sources:
  - name: "local-documents"
    type: "file"
    base_path: "./documents"
    file_pattern: "**/*.{pdf,docx,txt,md,json}"

logging:
  level: INFO
EOF
```

4. **Start Backend API:**
```bash
# Set environment variables
export GO_DOC_GO_CONFIG_PATH=config.yaml
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8000

# Start server
python -m go_doc_go.server

# Or use gunicorn for production
pip install gunicorn
gunicorn --bind 0.0.0.0:8000 --workers 4 go_doc_go.server:app
```

### Frontend Installation

1. **Install Node.js dependencies:**
```bash
cd frontend
npm install
```

2. **Configure frontend:**
```bash
# Create environment configuration
cat > .env.local << EOF
REACT_APP_API_URL=http://localhost:8000/api
REACT_APP_VERSION=1.0.0
EOF
```

3. **Development server:**
```bash
npm run dev
# Frontend available at http://localhost:5173
```

4. **Production build:**
```bash
# Build for production
npm run build

# Serve with nginx
sudo apt install nginx

# Nginx configuration
cat > /etc/nginx/sites-available/go-doc-go << EOF
server {
    listen 80;
    server_name your-domain.com;
    root /path/to/go-doc-go/frontend/dist;
    index index.html;

    # Frontend routes
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://localhost:8000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # Static assets
    location /static/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/go-doc-go /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### Worker Setup

1. **Install worker dependencies:**
```bash
# Already included with go-doc-go[all]
pip install "go-doc-go[embedding-all,source-all]"
```

2. **Configure worker:**
```bash
# Same configuration as backend
export GO_DOC_GO_CONFIG_PATH=config.yaml
export GO_DOC_GO_MODE=worker
export WORKER_ID=worker-$(hostname)-$$
```

3. **Start workers:**
```bash
# Start single worker
python -m go_doc_go.cli.worker

# Start multiple workers with supervisord
sudo apt install supervisor

# Supervisor configuration
cat > /etc/supervisor/conf.d/go-doc-go-workers.conf << EOF
[program:go-doc-go-worker]
command=/usr/local/bin/python -m go_doc_go.cli.worker
directory=/opt/go-doc-go
user=go-doc-go
environment=GO_DOC_GO_CONFIG_PATH=config.yaml,WORKER_ID=worker-%(process_num)d
numprocs=3
process_name=%(program_name)s-%(process_num)d
autostart=true
autorestart=true
stdout_logfile=/var/log/go-doc-go/worker-%(process_num)d.log
stderr_logfile=/var/log/go-doc-go/worker-%(process_num)d-error.log
EOF

sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start go-doc-go-worker:*
```

### Redis Installation (Optional)

```bash
# Install Redis
sudo apt install redis-server

# Configure Redis
sudo vim /etc/redis/redis.conf
# Set: bind 127.0.0.1
# Set: maxmemory 1gb
# Set: maxmemory-policy allkeys-lru

# Start Redis
sudo systemctl enable redis-server
sudo systemctl start redis-server

# Update Go-Doc-Go configuration
echo "redis_url: redis://localhost:6379" >> config.yaml
```

---

## Configuration Management

### Environment Variables

**Backend Server:**
```bash
export GO_DOC_GO_CONFIG_PATH=/path/to/config.yaml
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8000
export LOG_LEVEL=INFO
export CORS_ORIGINS=*
export API_KEY=your_api_key
export MAX_RESULTS=100
export REQUEST_TIMEOUT=30
```

**Database:**
```bash
export DATABASE_URL=postgresql://user:pass@host:port/dbname
export REDIS_URL=redis://localhost:6379
```

**Workers:**
```bash
export GO_DOC_GO_MODE=worker
export WORKER_ID=unique-worker-id
export WORKER_TIMEOUT=300
export MAX_MEMORY_MB=4096
export MAX_CPU_PERCENT=80
```

**API Keys:**
```bash
export OPENAI_API_KEY=your_openai_key
export ANTHROPIC_API_KEY=your_anthropic_key
```

### Configuration Files

**Main Configuration** (`config.yaml`):
```yaml
# Storage backend
storage:
  backend: postgresql
  path: postgresql://go_doc_go:password@localhost/go_doc_go

# Embedding configuration
embedding:
  enabled: true
  model: sentence-transformers/all-MiniLM-L6-v2
  dimensions: 384
  batch_size: 64

# Relationship detection
relationship_detection:
  enabled: true
  similarity_threshold: 0.7
  max_relationships_per_element: 5
  cross_document_semantic:
    similarity_threshold: 0.7

# Content sources
content_sources:
  - name: "documents"
    type: "file"
    base_path: "/data/documents"
    file_pattern: "**/*.{pdf,docx,txt,md,json,csv}"
    max_file_size: 100MB
    
  - name: "confluence"
    type: "confluence"
    url: "https://company.atlassian.net"
    username: "user@company.com"
    api_token: "${CONFLUENCE_API_TOKEN}"

# Domain detection (optional)
relationship_detection:
  domain:
    ontologies:
      - path: "examples/ontologies/financial_markets.yaml"
        active: true

# Logging
logging:
  level: INFO
  format: "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
```

**Redis Configuration** (`config/redis.conf`):
```conf
# Memory management
maxmemory 1gb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000

# Security
requirepass your_redis_password
bind 127.0.0.1

# Performance
tcp-keepalive 300
timeout 0
```

---

## Service Management

### Systemd Services (Linux)

**Backend Service** (`/etc/systemd/system/go-doc-go-backend.service`):
```ini
[Unit]
Description=Go-Doc-Go Backend API
After=network.target postgresql.service

[Service]
Type=exec
User=go-doc-go
Group=go-doc-go
WorkingDirectory=/opt/go-doc-go
Environment=GO_DOC_GO_CONFIG_PATH=/opt/go-doc-go/config.yaml
Environment=PYTHONPATH=/opt/go-doc-go
ExecStart=/usr/local/bin/gunicorn --bind 0.0.0.0:8000 --workers 4 go_doc_go.server:app
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Worker Service** (`/etc/systemd/system/go-doc-go-worker@.service`):
```ini
[Unit]
Description=Go-Doc-Go Worker %i
After=network.target postgresql.service go-doc-go-backend.service

[Service]
Type=exec
User=go-doc-go
Group=go-doc-go
WorkingDirectory=/opt/go-doc-go
Environment=GO_DOC_GO_CONFIG_PATH=/opt/go-doc-go/config.yaml
Environment=GO_DOC_GO_MODE=worker
Environment=WORKER_ID=worker-%i
Environment=PYTHONPATH=/opt/go-doc-go
ExecStart=/usr/local/bin/python -m go_doc_go.cli.worker
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Enable and start services:**
```bash
# Enable services
sudo systemctl enable go-doc-go-backend
sudo systemctl enable go-doc-go-worker@1
sudo systemctl enable go-doc-go-worker@2
sudo systemctl enable go-doc-go-worker@3

# Start services
sudo systemctl start go-doc-go-backend
sudo systemctl start go-doc-go-worker@{1,2,3}

# Check status
sudo systemctl status go-doc-go-backend
sudo systemctl status go-doc-go-worker@1
```

---

## Monitoring and Observability

### Prometheus Metrics

**Metrics Configuration** (`monitoring/prometheus.yml`):
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'go-doc-go-backend'
    static_configs:
      - targets: ['backend:8000']
    metrics_path: '/metrics'

  - job_name: 'go-doc-go-workers'
    static_configs:
      - targets: ['worker:9090']

  - job_name: 'postgresql'
    static_configs:
      - targets: ['postgres_exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis_exporter:9121']
```

### Grafana Dashboards

Access Grafana at `http://localhost:3001` with the credentials from your secrets.

**Key Dashboards:**
- System Overview (CPU, Memory, Disk)
- Document Processing Pipeline
- Search Performance
- Worker Activity
- Database Performance

### Log Aggregation

**Loki Configuration** (`monitoring/loki.yml`):
```yaml
auth_enabled: false

server:
  http_listen_port: 3100

ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

storage_config:
  boltdb_shipper:
    active_index_directory: /tmp/loki/boltdb-shipper-active
    cache_location: /tmp/loki/boltdb-shipper-cache
  filesystem:
    directory: /tmp/loki/chunks

limits_config:
  reject_old_samples: true
  reject_old_samples_max_age: 168h

chunk_store_config:
  max_look_back_period: 0s

table_manager:
  retention_deletes_enabled: false
  retention_period: 0s
```

---

## Security Configuration

### API Security

```bash
# Generate API key
export API_KEY=$(openssl rand -hex 32)

# Configure authentication
export API_KEY_HEADER=X-API-Key

# Enable rate limiting
export RATE_LIMIT="100 per minute"
```

### Database Security

```sql
-- PostgreSQL security
CREATE USER go_doc_go_readonly WITH ENCRYPTED PASSWORD 'readonly_password';
GRANT CONNECT ON DATABASE go_doc_go TO go_doc_go_readonly;
GRANT USAGE ON SCHEMA public TO go_doc_go_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO go_doc_go_readonly;

-- Create backup user
CREATE USER go_doc_go_backup WITH ENCRYPTED PASSWORD 'backup_password';
GRANT CONNECT ON DATABASE go_doc_go TO go_doc_go_backup;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO go_doc_go_backup;
```

### Firewall Configuration

```bash
# Ubuntu/Debian with UFW
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 22/tcp    # SSH

# Internal services (restrict to local network)
sudo ufw allow from 10.0.0.0/8 to any port 8000   # Backend API
sudo ufw allow from 10.0.0.0/8 to any port 5432   # PostgreSQL
sudo ufw allow from 10.0.0.0/8 to any port 6379   # Redis
```

---

## Production Deployment Checklist

### Pre-Deployment

- [ ] **System Requirements Met**: Memory, CPU, Storage
- [ ] **Dependencies Installed**: Python, Node.js, Database
- [ ] **Database Created**: Schema initialized
- [ ] **Configuration Files**: Reviewed and customized
- [ ] **Secrets Management**: API keys and passwords secured
- [ ] **SSL Certificates**: Obtained and configured (if HTTPS)
- [ ] **Firewall Rules**: Configured for security
- [ ] **Monitoring Setup**: Prometheus, Grafana configured
- [ ] **Backup Strategy**: Database backup configured

### Deployment

- [ ] **Build Images**: Docker images built and tagged
- [ ] **Deploy Database**: PostgreSQL/Redis deployed and running
- [ ] **Deploy Backend**: API server deployed and healthy
- [ ] **Deploy Workers**: Processing workers scaled appropriately
- [ ] **Deploy Frontend**: UI built and served by nginx
- [ ] **Configure Load Balancer**: If using multiple backend instances
- [ ] **SSL Configuration**: HTTPS enabled with valid certificates
- [ ] **DNS Configuration**: Domain pointing to correct servers

### Post-Deployment

- [ ] **Health Checks**: All services responding correctly
- [ ] **Performance Testing**: Load testing completed
- [ ] **Log Aggregation**: Logs being collected properly
- [ ] **Monitoring Alerts**: Alerting rules configured
- [ ] **Backup Testing**: Backup and restore procedures tested
- [ ] **Documentation**: Deployment documentation updated
- [ ] **Team Training**: Operations team trained on system

---

## Troubleshooting

### Common Issues

**Backend won't start:**
```bash
# Check configuration
python -c "from go_doc_go import Config; c = Config(); print('Config loaded successfully')"

# Check database connection
python -c "from go_doc_go import Config; c = Config(); db = c.get_document_database(); db.initialize()"

# Check logs
journalctl -u go-doc-go-backend -f
```

**Workers not processing:**
```bash
# Check worker registration
docker-compose exec backend python -c "
from go_doc_go.queue.work_queue import WorkQueue, RunCoordinator
from go_doc_go.storage.postgres import PostgreSQLDocumentDatabase
from tests.test_queue.test_db_adapter import QueueDatabaseAdapter

config = {'host': 'postgres', 'port': 5432, 'database': 'go_doc_go', 'user': 'go_doc_go', 'password': 'changeme'}
db = QueueDatabaseAdapter(PostgreSQLDocumentDatabase(config))
coordinator = RunCoordinator(db)
print('Active workers:', coordinator.get_active_workers())
"

# Check work queue
docker-compose exec backend python -c "
from go_doc_go.queue.work_queue import WorkQueue
queue = WorkQueue(db, 'debug')
print('Pending jobs:', queue.get_queue_status())
"
```

**Frontend not loading:**
```bash
# Check build files
ls -la frontend/dist/

# Check nginx configuration
nginx -t

# Check nginx logs
tail -f /var/log/nginx/error.log
```

**Database connection issues:**
```bash
# Test PostgreSQL connection
psql -h localhost -U go_doc_go -d go_doc_go -c "SELECT version();"

# Check PostgreSQL logs
docker-compose logs postgres

# Check connection limits
psql -h localhost -U go_doc_go -d go_doc_go -c "SHOW max_connections;"
```

### Performance Optimization

**Backend Performance:**
```bash
# Scale API servers
docker-compose up --scale backend=3

# Use connection pooling
pip install psycopg2[pool]
# Update DATABASE_URL to include pooling parameters
```

**Worker Performance:**
```bash
# Scale workers
docker-compose up --scale worker=5

# Adjust worker resources
export WORKER_MEMORY_LIMIT=8G
export WORKER_CPU_LIMIT=4.0
```

**Database Performance:**
```sql
-- PostgreSQL optimization
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
SELECT pg_reload_conf();

-- Create indexes for performance
CREATE INDEX CONCURRENTLY idx_elements_doc_id ON elements(doc_id);
CREATE INDEX CONCURRENTLY idx_embeddings_element_pk ON embeddings(element_pk);
```

---

## Support and Maintenance

### Regular Maintenance Tasks

**Daily:**
- Monitor system health and performance
- Check log files for errors
- Verify backup completion

**Weekly:**
- Review system metrics and trends
- Update security patches
- Clean up old log files
- Database maintenance (VACUUM, ANALYZE)

**Monthly:**
- Security audit
- Performance analysis
- Capacity planning review
- Documentation updates

### Getting Help

- **Documentation**: [Go-Doc-Go Documentation](../README.md)
- **Issues**: [GitHub Issues](https://github.com/kenstott/go-doc-go/issues)
- **API Reference**: http://your-server:8000/docs
- **Configuration Guide**: [Configuration Documentation](configuration.md)

### Version Upgrades

```bash
# Backup before upgrade
pg_dump go_doc_go > backup_$(date +%Y%m%d).sql

# Pull latest version
docker-compose pull

# Deploy with zero downtime
docker-compose up -d --no-deps backend
docker-compose up -d --no-deps worker

# Verify deployment
docker-compose ps
curl http://localhost:8000/health
```

---

This comprehensive guide covers all aspects of installing and deploying the Go-Doc-Go system from development to production. Choose the deployment method that best fits your environment and requirements.