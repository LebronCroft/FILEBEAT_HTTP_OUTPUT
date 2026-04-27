# Troubleshooting

## Build Failed

- Run `go mod download` to fetch dependencies.
- Confirm the local Go version is `1.21` or later.
- Run `gofmt -w .` and `go test ./...` to catch syntax or package issues.
- If dependency download fails, check network access and Go proxy configuration.

## Cannot Connect To Server

- Verify `output.http.hosts` uses the correct scheme, host, and port.
- Confirm the receiver is listening and reachable from the Filebeat host.
- Check firewall, proxy, and container network settings.
- If `proxy_url` is configured, test both direct and proxied connectivity.

## TLS Handshake Failed

- Use `https://` in `hosts`.
- Confirm `tls.certificate_authorities` points to the CA that signed the server certificate.
- For mutual TLS, verify `tls.certificate` and `tls.key` match.
- Check certificate expiration, hostname SANs, and file permissions.

## Logs Not Forwarded

- Confirm Filebeat inputs or modules are enabled.
- Check the module path under `filebeat.config.modules.path`.
- Confirm the HTTP receiver accepts `POST` requests at the configured `path`.
- Enable debug logging and inspect HTTP output errors.
- Check whether the receiver is returning `3xx`, `4xx`, or `5xx` responses.

## Filebeat Plugin Not Loaded

- Ensure the binary was built with the HTTP output package imported.
- Confirm the configuration uses `output.http`.
- Rebuild the binary after source changes.
- Check startup logs for unknown output type errors.

## High Memory Usage

- Reduce `queue.mem.events`.
- Reduce `batch_size`.
- Use `batch_publish: true` to reduce per-event overhead.
- Check whether the receiver is slow or unavailable, causing retries to accumulate.

## Retry Too Frequently

- Increase `backoff.init` and `backoff.max`.
- Increase `timeout` if the receiver is slow but healthy.
- Lower `max_retries` if events should fail faster.
- Inspect receiver status codes. Repeated `3xx` or unexpected `4xx` responses can cause retry pressure.
