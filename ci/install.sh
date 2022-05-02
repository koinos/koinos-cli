#!/bin/bash

sudo gem install coveralls-lcov
go install github.com/jandelgado/gcov2lcov@latest
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.45.2
