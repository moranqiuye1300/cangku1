$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"

New-Item -ItemType Directory -Force -Path "gen" | Out-Null

protoc -I proto `
  --go_out=gen --go_opt=module=short-video-platform/gen `
  --go-grpc_out=gen --go-grpc_opt=module=short-video-platform/gen `
  proto/video.proto proto/user.proto

Write-Host "proto generated under gen/userpb and gen/videopb"
