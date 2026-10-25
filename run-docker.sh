#!/bin/sh
# Runs gofmt, go vet, staticcheck, the tests and a build in the official Go
# image, so your local Go does not matter. Usage: ./run-docker.sh
set -e
docker run --rm -v "$PWD":/app -w /app golang:1.23 sh -c '
  set -e
  unformatted=$(gofmt -l .)
  if [ -n "$unformatted" ]; then echo "gofmt needed on: $unformatted"; exit 1; fi
  go vet ./...
  go install honnef.co/go/tools/cmd/staticcheck@2024.1.1
  staticcheck ./...
  go test -race -count=1 -cover ./...
  go build ./...
  echo "All checks passed."
'
