#!/usr/bin/env bash
# Generates gen/shredstream from the shipped schemas. Requires protoc plus:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
set -euo pipefail
cd "$(dirname "$0")"

# The vendored Jito protos carry no `option go_package` (nothing upstream needs one), so the Go
# package is supplied here with M<file>= mappings instead of editing a byte-pinned vendored file.
#
# ⚠ Two mappings, two packages: BOTH files declare a message named `Heartbeat`, so mapping them
# into one Go package makes `Heartbeat.ProtoReflect already declared` and the build dies.
MAP="Mshredstream.proto=deshred-entries-go/gen/shredstream,Mshared.proto=deshred-entries-go/gen/shared"

mkdir -p gen
protoc -I ../../proto/jito-shredstream \
  --go_out=. --go_opt=module=deshred-entries-go --go_opt="$MAP" \
  --go-grpc_out=. --go-grpc_opt=module=deshred-entries-go --go-grpc_opt="$MAP" \
  shredstream.proto shared.proto

go mod tidy
echo "generated: $(find gen -name '*.go' | sort | tr '\n' ' ')"
