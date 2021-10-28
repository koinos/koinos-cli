#!/bin/bash

set -e
set -x

go test -v github.com/koinos/koinos-cli-wallet/internal/cli -coverprofile=./build/cli.out -coverpkg=./koinos/cli
gcov2lcov -infile=./build/cli.out -outfile=./build/cli.info

golint -set_exit_status ./...
