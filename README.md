# Beats HTTP Output

Beats HTTP Output is a Filebeat-compatible custom output plugin that forwards Filebeat events to HTTP or HTTPS ingestion services.

The project is intended for teams that want to keep Filebeat's mature input, module, queue, and publishing pipeline while delivering events to a custom log platform instead of Elasticsearch.

## Why This Exists

Filebeat has strong support for collecting and parsing logs, but some environments need to send events to an internal API, a SaaS gateway, or a custom storage pipeline. This plugin adds an HTTP output path so Filebeat events can be serialized and posted to one or more HTTP endpoints.

## Features

- HTTP and HTTPS event forwarding.
- Multiple target hosts with optional Filebeat network-client load balancing.
- Batch or per-event publishing.
- JSON and JSON Lines payload formats.
- Optional gzip compression.
- Custom headers, query parameters, content type, basic authentication, and proxy URL.
- TLS configuration through Beats' `tls` settings.
- Retry and backoff behavior through the Beats output pipeline.
- Runtime wrapper support for module configuration reloads in this repository.

## Architecture Overview

Filebeat collects events from inputs or modules, enriches them through the Beats pipeline, and publishes batches to the registered `http` output. The output creates one HTTP client per configured host, serializes events, and sends `POST` requests to the configured endpoint.

Important paths:

- `libbeat/outputs/http`: HTTP output implementation.
- `config`: runtime configuration loading and module config generation helpers.
- `script`: optional module watcher and script manager.
- `filebeat`: Filebeat entry point integration.
- `modules.d`: example Filebeat module configuration.

See [docs/architecture.md](docs/architecture.md) for the detailed event flow.

## Quick Start

Build the project:

```bash
go build ./...
```

Run tests:

```bash
go test ./...
```

Prepare a configuration:

```bash
cp config.example.yml filebeat.yml
```

Update `output.http.hosts` and `output.http.path`, then run your Filebeat binary or wrapper with the configuration.

## Configuration Example

```yaml
filebeat:
  config:
    modules:
      enabled: true
      path: ./modules.d/filebeat.yml
      reload:
        enabled: true
        period: 10s

queue:
  mem:
    events: 10000
    flush:
      min_events: 500
      timeout: 5s

output:
  http:
    hosts:
      - "https://example.com:8080"
    path: "/receive/log"
    max_retries: 3
    timeout: 60s
    format: json
    batch_publish: true
    batch_size: 2048
    headers:
      Authorization: "Bearer your-token-here"
```

Full option documentation is available in [docs/configuration.md](docs/configuration.md).

## Build From Source

Requirements:

- Go 1.21 or later.
- Network access to download Go modules.

Commands:

```bash
go mod download
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

To build the Filebeat entry package only:

```bash
go build ./filebeat
```

## Usage With Filebeat

Registering the plugin happens through the blank import of `libbeat/outputs/http`. A Filebeat build that includes this repository can use:

```yaml
output:
  http:
    hosts:
      - "https://example.com:8080"
    path: "/receive/log"
```

The repository also contains a wrapper in `main.go` that expects a local `filebeatexc` executable and starts it with the generated `filebeat.yml`.

## Example Output

In `format: json` mode with batch publishing, the receiver gets an array of event objects:

```json
[
  {
    "@timestamp": "2026-01-01T00:00:00Z",
    "message": "service started",
    "log": {"level": "info"}
  }
]
```

In `format: json_lines` mode, events are written as newline-delimited JSON records.

## TLS Configuration

TLS is provided by Beats' `tls` configuration support:

```yaml
output:
  http:
    hosts:
      - "https://example.com:8080"
    path: "/receive/log"
    tls:
      certificate_authorities:
        - "your-cert.pem"
      certificate: "your-client-cert.pem"
      key: "your-key.pem"
```

Use `https://` hosts when TLS is required.

## Load Balancing Behavior

Set `loadbalance: true` to let Beats distribute publishing across all configured HTTP clients. When it is disabled, Beats uses the standard failover behavior for network outputs.

```yaml
output:
  http:
    hosts:
      - "https://example.com:8080"
      - "https://example.com:8081"
    loadbalance: true
```

## File Output Mode

This repository does not implement a separate `output.file` replacement. File collection is handled by Filebeat inputs and modules, and local operational logs are written through the wrapper logging configuration.

If durable local spooling or a dedicated file-output mode is required, track it as a future enhancement.

## HTTP Forwarding Mode

The HTTP output sends `POST` requests to the final URL created from:

- `hosts`
- `protocol`
- `path`
- `parameters`

The request body is encoded as JSON or JSON Lines, with optional gzip compression. Status codes `400` and `500` are treated as non-retryable by the current implementation; other `3xx` and higher responses are returned to the Beats retry pipeline.

## Performance Considerations

- Increase `batch_size` for higher throughput when the receiver accepts large payloads.
- Use `batch_publish: true` to reduce request count.
- Tune `queue.mem.events` and `queue.mem.flush` for bursty workloads.
- Compression can reduce network usage but increases CPU cost.
- Keep `timeout`, `backoff.init`, `backoff.max`, and `max_retries` aligned with receiver latency and availability.

## Troubleshooting

Start with [docs/troubleshooting.md](docs/troubleshooting.md). Common checks:

- Confirm the receiver is reachable from the Filebeat host.
- Verify the configured URL, path, proxy, and TLS certificates.
- Run `go test ./...` after source changes.
- Enable debug logging when diagnosing publish failures.

## Roadmap

- Add integration tests with a local HTTP receiver.
- Add clearer runtime metrics for HTTP status codes and retries.
- Document packaging workflows for custom Filebeat distributions.
- Consider a durable local file spool or file-output mode.
- Add release automation for GitHub tags and checksums.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## License

This project is released under the MIT License. Some files are derived from Elastic Beats and retain their original Apache 2.0 license headers; see [LICENSE.txt](LICENSE.txt) and [NOTICE.txt](NOTICE.txt) for upstream notices.
