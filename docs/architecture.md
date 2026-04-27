# Architecture

## Filebeat Event Flow

Filebeat reads log data from configured inputs or modules. Inputs create events, processors can enrich or filter those events, and the Beats publisher pipeline groups them into batches for an output.

In this repository, the HTTP output is registered as `output.http`. Once Filebeat selects that output, batches are delivered to the plugin through the Beats network output interface.

## Custom Output Plugin Flow

The plugin entry point is `MakeHTTP` in `libbeat/outputs/http/http.go`.

The flow is:

1. Unpack `output.http` configuration.
2. Load TLS settings with Beats' TLS helper.
3. Read the configured host list.
4. Parse proxy settings and query parameters.
5. Create one HTTP client per host.
6. Wrap each client with Beats backoff handling.
7. Return a Beats network output group with retry and load-balancing settings.

## HTTP Forwarding Process

The client receives a publisher batch and either sends the batch as one request or sends each event individually.

For each request:

1. Build the final URL from host, path, and query parameters.
2. Convert Filebeat events into JSON-compatible maps.
3. Encode the body as `json` or `json_lines`.
4. Optionally compress the body with gzip.
5. Add default, configured, and encoder-specific headers.
6. Send an HTTP `POST` request.
7. Acknowledge the batch or return failed events to the retry pipeline.

## Retry And Failure Handling

The output relies on Beats network-output retry handling. `max_retries`, `backoff.init`, and `backoff.max` control how retry attempts are scheduled.

Current response handling:

- Network errors mark the connection as disconnected and return the error.
- Status codes lower than `300` are treated as success.
- Status codes `400` and `500` are treated as non-retryable in the current implementation.
- Other status codes greater than or equal to `300` are returned as errors and can be retried by the Beats pipeline.
- JSON encoding failures are treated as non-retryable.

## TLS Transmission

TLS settings are passed through Beats' `tlscommon.LoadTLSConfig`. Use `https://` hosts and configure `tls.certificate_authorities`, `tls.certificate`, and `tls.key` when the receiver requires trusted CAs or mutual TLS.

The HTTP client uses Beats transport dialers for normal and TLS connections, with the configured request timeout applied to both dialing and request execution.

## Load Balancing Strategy

The plugin creates one client per configured host. When `loadbalance: true`, Beats distributes work across the available network clients. When `loadbalance: false`, Beats uses its standard failover behavior.

Use multiple hosts only when all receivers accept the same payload contract and authentication configuration.
