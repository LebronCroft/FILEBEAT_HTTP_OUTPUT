# Public Release Checklist

This repository contains custom work on top of Beats with an HTTP forwarding output and extra local logging.

## Remove or replace before publishing

- Replace private endpoints in configuration samples.
- Remove IDE metadata such as `.idea/`, especially any SSH or deployment settings.
- Remove temporary logs, binaries, and local caches.
- Review shell commands and scripts for hard-coded local paths.
- Review comments and docs for company-specific names or internal environment references.

## Items found in this repository

- `filebeat.yml` contained a private-looking HTTP endpoint and has been replaced with a safe example.
- `.idea/sshConfigs.xml` contains an SSH host and username.
- `.idea/webServers.xml` contains an SFTP host and username.
- `README.md` originally pointed to another maintainer's GitHub repository and has been rewritten for personal publishing.
- `go.mod` and Go import paths still use `github.com/fufuok/beats-http-output`.

## Before first GitHub push

1. Pick your final repository name.
2. Update the module path in `go.mod`.
3. Update internal imports from the old module path to the new one.
4. Remove already tracked IDE files from Git history or at least from the next commit.
5. Re-run a repository-wide search for IPs, emails, usernames, and tokens.

## Suggested search

```powershell
rg -n --hidden -S "172\\.|10\\.|192\\.168\\.|password|token|secret|username|ssh|sftp|proxy_url|fufuok|jpchev|电信" .
```
