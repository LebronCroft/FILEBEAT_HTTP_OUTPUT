# Contributing

Thanks for your interest in contributing to Beats HTTP Output.

## Development Workflow

1. Fork the repository and create a feature branch.
2. Keep changes focused and avoid unrelated refactors.
3. Run formatting and checks before opening a pull request:

```bash
gofmt -w .
go vet ./...
go test ./...
```

4. Update documentation or examples when configuration or behavior changes.
5. Open a pull request with a clear summary, test notes, and any compatibility considerations.

## Pull Request Guidelines

- Preserve existing Filebeat and Beats integration behavior.
- Do not commit private configuration, certificates, tokens, logs, or generated binaries.
- Add tests for behavior changes when practical.
- Keep public examples generic and runnable with placeholder endpoints.

## Reporting Bugs

Please include:

- Go version.
- Operating system.
- Filebeat or Beats version.
- Relevant configuration with secrets removed.
- Error logs and reproduction steps.
