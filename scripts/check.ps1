param(
    [string]$GoCache = ".gocache",
    [string]$GoModCache = ".gomodcache"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $GoCache, $GoModCache | Out-Null

$env:GOCACHE = (Resolve-Path $GoCache).Path
$env:GOMODCACHE = (Resolve-Path $GoModCache).Path

gofmt -w main.go test.go
Get-ChildItem -Path config,infra,libbeat\outputs\http,script,filebeat,enum -Recurse -Include *.go |
    ForEach-Object { gofmt -w $_.FullName }

go vet ./...
go test ./...
