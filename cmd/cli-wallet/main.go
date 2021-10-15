package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli-wallet/cmd/cli-wallet/interactive"
	"github.com/koinos/koinos-cli-wallet/internal/wallet"
	flag "github.com/spf13/pflag"
)

// Commpand line parameter names
const (
	rpcOption     = "rpc"
	executeOption = "execute"
	versionOption = "version"
)

// Default options
const (
	rpcDefault = ""
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	// Setup command line options
	rpcAddress := flag.StringP(rpcOption, "r", rpcDefault, "RPC server URL")
	executeCmd := flag.StringSliceP(executeOption, "x", nil, "Command to execute")
	versionCmd := flag.BoolP(versionOption, "v", false, "Display the version")

	flag.Parse()

	if *versionCmd {
		fmt.Println(wallet.Version)
		os.Exit(0)
	}

	// Setup client
	var client *wallet.KoinosRPCClient
	if *rpcAddress != "" {
		client = wallet.NewKoinosRPCClient(*rpcAddress)
	}

	// Construct the command parser
	commands := wallet.NewKoinosCommandSet()
	parser := wallet.NewCommandParser(commands)

	cmdEnv := wallet.NewExecutionEnvironment(client, parser)

	// If the user submitted commands, execute them
	if *executeCmd != nil {
		for _, cmd := range *executeCmd {
			results := wallet.ParseAndInterpret(parser, cmdEnv, cmd)
			results.Print()
		}
		// Otherwise run the interactive shell
	} else {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, cmdEnv)
		p.Run()
	}
}
