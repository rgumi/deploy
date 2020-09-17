#!/bin/sh

cd src
go test ./... -coverprofile="coverage-report.out"
go test ./... -json > test-report.out
GOOS=linux GOARCH=amd64 go build 