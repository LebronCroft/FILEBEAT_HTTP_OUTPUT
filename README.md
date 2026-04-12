# Beats HTTP Forwarder

Beats HTTP Forwarder is a log collection and forwarding project for sending events to custom HTTP services.

It is designed for scenarios where logs need to be collected from files or modules, normalized through the Beats pipeline, and then delivered to your own HTTP endpoint for downstream processing, storage, or analysis.

## Features

- Collect logs with Filebeat-style configuration.
- Forward events to a custom HTTP endpoint.
- Support configurable request path, retry policy, timeout, and headers.
- Support JSON and JSON Lines payload formats.
- Support optional gzip compression.
- Provide local log output for troubleshooting runtime behavior.

## Use cases

- Forward server logs to an internal log platform.
- Send collected events to a custom ingestion API.
- Replace or supplement Elasticsearch output with HTTP delivery.
- Build a lightweight log shipping bridge between edge machines and a central service.

## Project structure

- `libbeat/outputs/http`: HTTP output implementation.
- `infra`: local logging utilities.
- `config`: configuration generation helpers.
- `filebeat.yml`: example runtime configuration.

## Requirements

- Go 1.21 or later
- A reachable HTTP service to receive log events

## Build

```bash
go mod tidy
go build .
```

If you want to build the Filebeat entry specifically:

```bash
cd filebeat
go build .
```

## Configuration

Example `filebeat.yml`:

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
      - "http://example.com:8080"
    path: "/receive/log"
    max_retries: 3
    timeout: 60s
```

## HTTP output options

- `hosts`: target HTTP server addresses.
- `path`: request path appended to the host.
- `max_retries`: retry count when delivery fails.
- `timeout`: HTTP request timeout.
- `headers`: custom request headers.
- `content_type`: custom request content type.
- `compression_level`: gzip compression level from `0` to `9`.
- `format`: payload format, supports `json` and `json_lines`.
- `proxy_url`: optional proxy server.
- `loadbalance`: enable load balancing across multiple hosts.

## How it works

1. The agent reads logs from configured inputs or modules.
2. Events are converted into structured payloads.
3. The HTTP output packages events as JSON or JSON Lines.
4. The payload is sent to the configured HTTP endpoint.
5. Retry and timeout settings control delivery behavior when errors occur.

## Run

After preparing the configuration, start the program in your preferred way for the target binary you build.

If your workflow uses the local executable wrapper in this repository, make sure the runtime configuration file and executable are present in the working directory.

## Notes

- Replace the example HTTP address with your real endpoint before running.
- Review `LICENSE.txt` and `NOTICE.txt` before public redistribution.
- Do not commit local IDE files, runtime logs, or private deployment configuration.
