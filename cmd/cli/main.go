package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli/cmd/cli/interactive"
	"github.com/koinos/koinos-cli/internal/cli"
	"github.com/koinos/koinos-cli/internal/cliutil"
	util "github.com/koinos/koinos-util-golang"
	flag "github.com/spf13/pflag"
)

// Commpand line parameter names
const (
	rpcOption              = "rpc"
	executeOption          = "execute"
	fileOption             = "file"
	versionOption          = "version"
	forceInteractiveOption = "force-interactive"
	forceTextPromptOption  = "force-text-prompt"
)

// Default options
const (
	rpcDefault = ""
)

// Other constants
const (
	rcFileName = ".koinosrc"
)

func main() {
	// Optionally load .env file
	_ = godotenv.Load()

	// Setup command line options
	rpcAddress := flag.StringP(rpcOption, "r", rpcDefault, "RPC server URL")
	executeCmd := flag.StringSliceP(executeOption, "x", nil, "Command to execute")
	fileCmd := flag.StringSliceP(fileOption, "f", nil, "File to execute")
	versionCmd := flag.BoolP(versionOption, "v", false, "Display the version")
	forceInteractive := flag.BoolP(forceInteractiveOption, "i", false, "Forces interactive mode. Useful for forcing a prompt when using the excute option")
	forceTextPrompt := flag.BoolP(forceTextPromptOption, "t", false, "Forces text prompt in interactive mode, rather than unicode symbols")

	flag.Parse()

	if *versionCmd {
		fmt.Println(cliutil.Version)
		os.Exit(0)
	}

	// Setup client
	var client *cliutil.KoinosRPCClient
	if *rpcAddress != "" {
		client = cliutil.NewKoinosRPCClient(*rpcAddress)
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

	// Create list of files to execute, intialize with rc files
	files := []string{path.Join(util.GetHomeDir(), rcFileName), rcFileName}

	if *fileCmd != nil {
		files = append(files, *fileCmd...)
	}

	for _, file := range files {
		// Make sure file exists, silently skip if not
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		results := make([]string, 0)

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			ir := cli.ParseAndInterpret(parser, cmdEnv, line)
			results = append(results, ir.Results...)
		}

		for _, result := range results {
			fmt.Println(result)
		}

		if len(results) > 0 {
			fmt.Println()
		}
	}

	// Run interactive mode if no commands given, or if forced
	if *forceInteractive || (*executeCmd == nil && *fileCmd == nil) {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, cmdEnv, *forceTextPrompt)
		p.Run()
	}
}
