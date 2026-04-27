# Configuration

This document describes the main configuration fields used by the repository and the custom `output.http` plugin.

## Filebeat Module Configuration

```yaml
filebeat:
  config:
    modules:
      enabled: true
      path: ./modules.d/filebeat.yml
      reload:
        enabled: true
        period: 10s
```

- `filebeat.config.modules.enabled`: Enables Filebeat module configuration loading.
- `filebeat.config.modules.path`: Path to the module configuration file used by this wrapper.
- `filebeat.config.modules.reload.enabled`: Enables module configuration reload.
- `filebeat.config.modules.reload.period`: Reload interval.

## Queue Configuration

- `queue.mem.events`: Maximum number of events kept in the in-memory queue.
- `queue.mem.flush.min_events`: Minimum events to trigger a flush.
- `queue.mem.flush.timeout`: Maximum wait before flushing a partial batch.

## HTTP Output

```yaml
output:
  http:
    hosts:
      - "https://example.com:8080"
    path: "/receive/log"
```

- `hosts`: List of target HTTP or HTTPS hosts.
- `protocol`: Optional protocol used when hosts do not include one.
- `path`: Request path appended to each host.
- `parameters`: Query parameters appended to the request URL.
- `username`: Basic authentication username.
- `password`: Basic authentication password.
- `proxy_url`: HTTP proxy URL.
- `loadbalance`: Enables Beats load balancing across configured hosts.
- `batch_publish`: Sends a whole batch in one request when `true`; sends one request per event when `false`.
- `batch_size`: Batch size passed to the Beats network output group.
- `compression_level`: Gzip compression level from `0` to `9`. `0` disables compression.
- `tls`: Beats TLS configuration for HTTPS and mutual TLS.
- `max_retries`: Maximum retry attempts controlled by the Beats output pipeline.
- `timeout`: HTTP client timeout.
- `headers`: Custom request headers.
- `content_type`: Overrides the request content type.
- `backoff.init`: Initial retry backoff duration.
- `backoff.max`: Maximum retry backoff duration.
- `format`: Payload format. Supported values are `json` and `json_lines`.

## Runtime Logging

```yaml
logging:
  path: ./logs
  module_name: filebeat-http-output
  max_size: 100
  max_backups: 3
  max_age: 30
  compress: true
```

- `logging.path`: Directory for wrapper logs.
- `logging.module_name`: Prefix for log files.
- `logging.max_size`: Maximum log file size in megabytes before rotation.
- `logging.max_backups`: Number of rotated log files to keep.
- `logging.max_age`: Number of days to keep rotated logs.
- `logging.compress`: Compresses rotated logs.

## Module Watcher

```yaml
scripts:
  module_watcher:
    enabled: true
    directory: ./modules.d
    interval: 1m
```

- `scripts.module_watcher.enabled`: Enables the module watcher.
- `scripts.module_watcher.directory`: Directory watched for module configuration changes.
- `scripts.module_watcher.interval`: Polling interval.
