package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli/cmd/cli/interactive"
	"github.com/koinos/koinos-cli/internal/cli"
	"github.com/koinos/koinos-cli/internal/kjsonrpc"
	"github.com/koinos/koinos-cli/internal/util"
	flag "github.com/spf13/pflag"
)

// Commpand line parameter names
const (
	rpcOption              = "rpc"
	executeOption          = "execute"
	versionOption          = "version"
	forceInteractiveOption = "force-interactive"
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
	forceInteractive := flag.BoolP(forceInteractiveOption, "i", false, "Forces interactive mode. Useful for forcing a prompt when using the excute option")

	flag.Parse()

	if *versionCmd {
		fmt.Println(util.Version)
		os.Exit(0)
	}

	// Setup client
	var client *kjsonrpc.KoinosRPCClient
	if *rpcAddress != "" {
		client = kjsonrpc.NewKoinosRPCClient(*rpcAddress)
	}

	// Construct the command parser
	commands := cli.NewKoinosCommandSet()
	parser := cli.NewCommandParser(commands)

	cmdEnv := cli.NewExecutionEnvironment(client, parser)

	// If the user submitted commands, execute them
	if *executeCmd != nil {
		for _, cmd := range *executeCmd {
			results := cli.ParseAndInterpret(parser, cmdEnv, cmd)
			results.Print()
		}
	}

	// Run interactive mode if no commands given, or if forced
	if *forceInteractive || (*executeCmd == nil) {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, cmdEnv)
		p.Run()
	}
}
