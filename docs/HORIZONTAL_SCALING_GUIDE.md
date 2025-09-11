# Horizontal Scaling Guide for Document Ingesters

This guide provides comprehensive instructions for deploying and horizontally scaling the distributed document processing system.

## Architecture Overview

The Go-Doc-Go distributed processing system uses an **elected leader architecture** where:
- Each process instance is an identical worker
- One worker is automatically elected as the leader for each processing run
- The elected leader handles document discovery and post-processing
- All workers (including leader) pull and process documents from the shared queue
- Leader election uses PostgreSQL for atomic coordination

## Prerequisites

### Infrastructure Requirements
- **PostgreSQL Database**: Shared queue storage (v12+ recommended)
- **Container Platform**: Docker/Kubernetes for orchestration
- **Shared Storage**: For binary document access (S3, NFS, etc.)
- **Monitoring**: Prometheus/Grafana (optional but recommended)

### Minimum Specifications
- **Single Worker**: 2 CPU cores, 4GB RAM
- **Database**: 4 CPU cores, 8GB RAM, SSD storage
- **Network**: Low latency between workers and database (<10ms recommended)

## Deployment Configurations

### 1. Single Process (Development/Small Scale)
```yaml
# config.yaml
processing:
  mode: "single"  # Traditional single-process mode

storage:
  backend: "sqlite"
  path: "./documents.db"
```

### 2. Distributed Processing (Production Scale)
```yaml
# config.yaml
processing:
  mode: "distributed"  # Coordinator + workers

work_queue:
  claim_timeout: 300      # 5 minutes per document
  heartbeat_interval: 30  # Worker heartbeat frequency  
  max_retries: 3         # Retry failed documents 3 times
  stale_threshold: 600   # 10 minutes to reclaim stale work

storage:
  backend: "postgresql"
  host: "postgres.example.com"
  port: 5432
  database: "go_doc_go"
  user: "processor_user"
  password: "${DB_PASSWORD}"
```

## Scaling Strategies

### Manual Scaling with Docker

All instances run the same command - they are identical workers that automatically elect a leader:

```bash
# Terminal 1: Start worker 1 (may become leader)
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e GO_DOC_GO_CONFIG_PATH=/app/config.yaml \
  go-doc-go:latest \
  python -m go_doc_go.cli.worker --worker-id worker-01

# Terminal 2: Start worker 2 (automatically joins run)
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e GO_DOC_GO_CONFIG_PATH=/app/config.yaml \
  go-doc-go:latest \
  python -m go_doc_go.cli.worker --worker-id worker-02

# Terminal 3: Start worker 3 (automatically joins run)
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e GO_DOC_GO_CONFIG_PATH=/app/config.yaml \
  go-doc-go:latest \
  python -m go_doc_go.cli.worker --worker-id worker-03

# Add more identical workers as needed...
```

### Kubernetes Deployment

#### ConfigMap for Shared Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: go-doc-go-config
data:
  config.yaml: |
    processing:
      mode: "distributed"
    work_queue:
      claim_timeout: 300
      heartbeat_interval: 30
      max_retries: 3
      stale_threshold: 600
    storage:
      backend: "postgresql"
      host: "postgres-service"
      port: 5432
      database: "go_doc_go"
      user: "processor_user"
      password_env: "DB_PASSWORD"
    content_sources:
      - name: "main-docs"
        type: "s3"
        bucket: "document-bucket"
        # ... source config
```

#### Worker Deployment (All Identical - Leader Election Automatic)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-doc-go-workers
spec:
  replicas: 5  # Scale this based on workload - all workers are identical
  selector:
    matchLabels:
      app: go-doc-go-worker
  template:
    metadata:
      labels:
        app: go-doc-go-worker
    spec:
      containers:
      - name: worker
        image: go-doc-go:latest
        command: ["python", "-m", "go_doc_go.cli.worker"]
        env:
        - name: GO_DOC_GO_CONFIG_PATH
          value: "/app/config.yaml"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: WORKER_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name  # Each pod gets unique worker ID
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        livenessProbe:
          exec:
            command:
            - python
            - -c
            - "import psutil; exit(0 if psutil.cpu_percent() < 90 else 1)"
          initialDelaySeconds: 60
          periodSeconds: 30
      volumes:
      - name: config
        configMap:
          name: go-doc-go-config
```

#### Horizontal Pod Autoscaler
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: go-doc-go-workers-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: go-doc-go-workers
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Scaling Best Practices

### 1. Performance Optimization

#### Database Connection Pooling
```yaml
# Add to config.yaml
storage:
  backend: "postgresql"
  # ... connection details
  pool_size: 10
  max_overflow: 20
  pool_timeout: 30
  pool_recycle: 3600
```

#### Worker Resource Allocation
- **CPU**: 1-2 cores per worker for most document types
- **Memory**: 2-4GB per worker (varies by document size)
- **I/O**: SSD storage for temporary files recommended

#### Database Sizing
- **Connections**: Plan for 5-10 connections per worker
- **Storage**: Queue tables are lightweight (~1KB per document)
- **Performance**: Use connection pooling and read replicas if needed

### 2. Monitoring and Observability

#### Built-in Monitoring
```bash
# Monitor processing in real-time
python -m go_doc_go.cli.monitor --run-id <run_id> --live

# Check dead letter queue
python -m go_doc_go.cli.deadletter --list --details

# Export metrics for external systems
python -m go_doc_go.cli.monitor --run-id <run_id> --export metrics.json
```

#### Prometheus Integration
```yaml
# Add to monitoring stack
- job_name: 'go-doc-go-metrics'
  static_configs:
  - targets: ['go-doc-go-coordinator:8080']
  scrape_interval: 30s
  metrics_path: '/metrics'
```

### 3. Failure Handling

#### Dead Letter Queue Management
```bash
# Analyze failure patterns
python -m go_doc_go.cli.deadletter --analyze

# Bulk retry failures after fixing issues
python -m go_doc_go.cli.deadletter --retry-run <run_id>

# Clean up old failures
python -m go_doc_go.cli.deadletter --purge 30
```

#### Worker Health Checks
```bash
# Check worker heartbeats
python -m go_doc_go.cli.monitor --run-id <run_id> --details

# Identify stale workers
python -m go_doc_go.cli.monitor --run-id <run_id> --alerts
```

## Scaling Guidelines

### Small Scale (< 10,000 documents)
- **Workers**: 2-5 instances
- **Database**: Single PostgreSQL instance
- **Monitoring**: Built-in CLI tools sufficient

### Medium Scale (10,000 - 100,000 documents)  
- **Workers**: 5-15 instances
- **Database**: PostgreSQL with connection pooling
- **Monitoring**: Prometheus + Grafana recommended
- **Dead Letter Queue**: Regular maintenance needed

### Large Scale (> 100,000 documents)
- **Workers**: 15-50+ instances  
- **Database**: PostgreSQL cluster with read replicas
- **Monitoring**: Full observability stack required
- **Dead Letter Queue**: Automated failure analysis
- **Network**: Dedicated network for database traffic

## Troubleshooting

### Common Issues

#### 1. Database Connection Exhaustion
**Symptoms**: Workers fail with connection errors
**Solution**: 
- Increase PostgreSQL `max_connections`
- Add connection pooling (pgbouncer)
- Reduce worker count temporarily

#### 2. Memory Issues with Large Documents
**Symptoms**: Workers crash with OOM errors
**Solution**:
- Increase worker memory limits
- Implement document size limits
- Use streaming parsers for large files

#### 3. Stale Work Items
**Symptoms**: Documents stuck in "processing" state
**Solution**:
- Check worker health and restart failed instances
- Reduce `claim_timeout` for faster recovery
- Monitor worker heartbeats

#### 4. Cross-Document Relationships Missing
**Symptoms**: Post-processing doesn't run or fails
**Solution**:
- Check leader election logs - ensure a leader was elected
- Verify leader lease renewal is working
- Check post-processing coordination logs
- Verify embedding configuration is correct

### Performance Tuning

#### Queue Performance
```sql
-- Add indexes for better queue performance
CREATE INDEX CONCURRENTLY idx_document_queue_run_status 
ON document_queue (run_id, status) 
WHERE status IN ('pending', 'processing');

CREATE INDEX CONCURRENTLY idx_document_queue_worker_heartbeat 
ON document_queue (worker_id, last_heartbeat) 
WHERE status = 'processing';
```

#### Worker Optimization
```yaml
# Optimize worker configuration
work_queue:
  claim_timeout: 180        # Reduce for faster failover
  heartbeat_interval: 15    # Increase frequency for better monitoring
  batch_size: 1            # Process one document at a time
  prefetch_count: 0        # No prefetching to prevent memory issues
```

## Security Considerations

### Database Security
- Use dedicated database user with minimal privileges
- Enable SSL/TLS for database connections
- Implement connection encryption
- Regular security updates for PostgreSQL

### Worker Security  
- Run workers with non-root user
- Limit file system access to necessary directories
- Use secrets management for credentials
- Regular security updates for base images

### Network Security
- Use private networks for database communication
- Implement network segmentation
- Monitor network traffic patterns
- Use VPN or service mesh for worker communication

## Deployment Checklist

### Pre-Deployment
- [ ] PostgreSQL cluster configured and tested
- [ ] Database migrations applied
- [ ] Connection pooling configured
- [ ] Monitoring stack deployed
- [ ] Security policies implemented
- [ ] Resource limits defined

### Initial Deployment
- [ ] Deploy initial worker set (2-3 identical instances)
- [ ] Verify leader election working (check logs for elected leader)
- [ ] Verify queue operations working
- [ ] Test document processing end-to-end
- [ ] Validate monitoring and alerting
- [ ] Test leader failover scenarios

### Production Scaling
- [ ] Implement horizontal pod autoscaling
- [ ] Configure load balancing if needed
- [ ] Set up automated dead letter queue maintenance
- [ ] Implement backup and disaster recovery
- [ ] Configure log aggregation and analysis
- [ ] Set up performance monitoring and alerting

This guide provides a comprehensive foundation for horizontally scaling the document processing system from development through large-scale production deployments.