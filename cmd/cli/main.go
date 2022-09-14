package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli/cmd/cli/interactive"
	"github.com/koinos/koinos-cli/internal/cli"
	"github.com/koinos/koinos-cli/internal/cliutil"
	"github.com/koinos/koinos-util-golang/rpc"
	flag "github.com/spf13/pflag"
)

// Commpand line parameter names
const (
	rpcOption              = "rpc"
	executeOption          = "execute"
	fileOption             = "file"
	versionOption          = "version"
	forceInteractiveOption = "force-interactive"
	rcFileOption           = "rc"
)

// Default options
const (
	rpcDefault    = ""
	rcFileDefault = ".koinosrc"
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
	fileCmd := flag.StringSliceP(fileOption, "f", nil, "File to execute")
	versionCmd := flag.BoolP(versionOption, "v", false, "Display the version")
	forceInteractive := flag.BoolP(forceInteractiveOption, "i", false, "Forces interactive mode. Useful for forcing a prompt when using the excute option")
	rcFile := flag.StringP(rcFileOption, "R", rcFileDefault, "Runs an rc file before dropping to interactive mode. Equivalent to --file and --force-interactive combined. (default: .koinosrc)")

	flag.Parse()

	if *versionCmd {
		fmt.Println(cliutil.Version)
		os.Exit(0)
	}

	// Setup client
	var client *rpc.KoinosRPCClient
	if *rpcAddress != "" {
		client = rpc.NewKoinosRPCClient(*rpcAddress)
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

	files := make([]string, 0)

	if len(*rcFile) != 0 {
		files = append(files, *rcFile)
	}

	if *fileCmd != nil {
		files = append(files, *fileCmd...)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			results := cli.ParseAndInterpret(parser, cmdEnv, line)
			results.Print()
		}
	}

	// Run interactive mode if no commands given, or if forced
	if *forceInteractive || (*executeCmd == nil && *fileCmd == nil) {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, cmdEnv)
		p.Run()
	}
}
