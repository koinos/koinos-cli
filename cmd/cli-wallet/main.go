package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli-wallet/internal"
	flag "github.com/spf13/pflag"
	"github.com/ybbus/jsonrpc/v2"
)

// Commpand line parameter names
const (
	rpcOption = "rpc"
)

// Default options
const (
	rpcDefault = "http://localhost:8080"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	// Setup command line options
	rpcAddress := flag.StringP(rpcOption, "r", rpcDefault, "RPC server URL")

	flag.Parse()

	// Setup command execution environment
	client := jsonrpc.NewClient(*rpcAddress)
	cmdEnv := internal.ExecutionEnvironment{RPCClient: &client}

	bcmd := internal.BalanceCommand{}
	bcmd.Execute(context.Background(), &cmdEnv)
}
