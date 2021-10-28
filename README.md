# Koinos CLI

Command-line interface (cli) for the Koinos blockchain

## Basic Usage:

When running the cli, it will start in interactive mode. Press tab or type `list` to see a list of possible commands.

`help <command-name>` will show a help message for the given command.

Some commands require a node rpc endpoint. This can be specified either when starting the cli with '--rpc' command line switch, or with the `connect` command from within the cli. Both take an endpoint url.

A circle top the left of the prompt will show red or green depending on whether or not you have specified an RPC endpoint.

`exit` or `quit` will quit the cli.

## Wallet creation / management

The lock symbol to the left of the prompt show whether or not you have a wallet open. Some commands require an open wallet.

To create a new wallet, use the command `create <filename> <password>`. The new wallet will then be created in the given file, and automatically opened.

To open a previously created wallet, use the command `open <filename> <password>`.

To import an existing WIF private key, use the commands `import <wif> <filename> <password>`.

To close an open wallet, simply use the `close` command.

Any of the commands which take a password may be called with it omitted. In this case it will use the value in the `WALLET_PASS` environment variable / .env file.

## Other useful commands

To check the balance of a given public address, use the commands `balance <address>`.

To transfer tKOIN from the currently open wallet, use the commands `transfer <amount> <address>`.

## Smart contract management

Note: Smart contract management will change in the future to be much easier to work with.

To upload a smart contract, use the command `upload <filename>`. The file given should be a compiled wasm smart contract. The contract id will be the public address of the currently open wallet.

To read from a smart contract, use the command `read <contract-id> <entry-point> <arguments>`. Entry-point should be a hex value such as 0x0D, as defined in the contract. Arguments should be a base64 string representing the binary arguments the entry-point requires.

To call a smart contract, use the command `call <contract-id> <entry-point> <arguments>`. The parameters here are given the same way as in the read command described above.

## Non-interactive mode

Commands can be executed without using interactive mode. The `--execute` command-line parameter takes a semicolon separated list of commands, executes them, then returns to the terminal.

## Building from source

To build the cli from source, you will need Go version 1.15 or higher.

From the root of the repository, simply run the command `go build -o koinos-cli cmd/cli/main.go`

## Public RPC server

There is a public RPC server that may be used for testing at this address: http://192.241.131.189:8080