package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli-wallet/cmd/cli-wallet/interactive"
	"github.com/koinos/koinos-cli-wallet/internal/wallet"
	flag "github.com/spf13/pflag"
)

// Commpand line parameter names
const (
	rpcOption     = "rpc"
	executeOption = "execute"
)

// Default options
const (
	rpcDefault     = ""
	executeDefault = ""
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	// Setup command line options
	rpcAddress := flag.StringP(rpcOption, "r", rpcDefault, "RPC server URL")
	executeCmd := flag.StringP(executeOption, "x", executeDefault, "Command to execute")

	flag.Parse()

	// Setup client
	var client *wallet.KoinosRPCClient
	if *rpcAddress != "" {
		client = wallet.NewKoinosRPCClient(*rpcAddress)
	}

	// Construct the command parser
	commands := wallet.NewKoinosCommandSet()
	parser := wallet.NewCommandParser(commands)

	cmdEnv := wallet.ExecutionEnvironment{RPCClient: client, Parser: parser}

	// If the user submitted commands, execute them
	if *executeCmd != "" {
		results := wallet.ParseAndInterpret(parser, &cmdEnv, *executeCmd)
		results.Print()
		// Otherwise run the interactive shell
	} else {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, &cmdEnv)
		p.Run()
	}
}
