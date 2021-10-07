#!/bin/bash

set -e
set -x

go test -v github.com/koinos/koinos-cli-wallet/internal/wallet -coverprofile=./build/wallet.out -coverpkg=./koinos/wallet
gcov2lcov -infile=./build/wallet.out -outfile=./build/wallet.info

golint -set_exit_status ./...
