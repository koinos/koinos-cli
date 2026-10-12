package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli/cmd/cli/interactive"
	"github.com/koinos/koinos-cli/internal/cli"
	"github.com/koinos/koinos-cli/internal/cliutil"
	util "github.com/koinos/koinos-util-golang/v2"
	flag "github.com/spf13/pflag"
)

// Command line parameter names
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

func main() {
	os.Exit(run(os.Args[1:], util.GetHomeDir()))
}

func run(args []string, homeDir string) int {
	flags := flag.NewFlagSet("koinos-cli", flag.ContinueOnError)
	// Optionally load .env file
	_ = godotenv.Load()

	// Setup command line options
	rpcAddress := flags.StringP(rpcOption, "r", rpcDefault, "RPC server URL (takes precedence over automatic startup files)")
	network := flags.String("network", "", "Select mainnet or testnet and load its startup files")
	executeCmd := flags.StringSliceP(executeOption, "x", nil, "Command to execute")
	fileCmd := flags.StringSliceP(fileOption, "f", nil, "File to execute")
	versionCmd := flags.BoolP(versionOption, "v", false, "Display the version")
	forceInteractive := flags.BoolP(forceInteractiveOption, "i", false, "Forces interactive mode after successfully executing commands and files")
	forceTextPrompt := flags.BoolP(forceTextPromptOption, "t", false, "Compatibility option; the prompt now always uses plain text")
	noRC := flags.Bool("no-rc", false, "Skip automatic home and current-directory startup files")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if *versionCmd {
		fmt.Println(cliutil.Version)
		return 0
	}

	if flags.Changed("network") && (!cli.IsNetwork(*network) || flags.Changed(rpcOption)) {
		fmt.Fprintln(os.Stderr, "choose --network mainnet or --network testnet, or supply --rpc separately")
		return 2
	}
	if flags.Changed(rpcOption) && *rpcAddress == "" {
		fmt.Fprintln(os.Stderr, "--rpc requires a node URL")
		return 2
	}

	// Construct the command parser
	commands := cli.NewKoinosCommandSet()
	parser := cli.NewCommandParser(commands)

	cmdEnv := cli.NewExecutionEnvironment(nil, parser)
	cmdEnv.PasswordPrompt = interactive.PasswordPrompt()
	target := *rpcAddress
	if *network != "" {
		target = *network
	}
	if target != "" {
		if err := cmdEnv.Connect(context.Background(), target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	// Register startup commands before parsing commands supplied with --execute.
	if !*noRC {
		if err := executeStartupFiles(homeDir, parser, cmdEnv, target != ""); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	// If the user submitted commands, execute them
	if *executeCmd != nil {
		for _, cmd := range *executeCmd {
			results := cli.ParseAndInterpret(parser, cmdEnv, cmd)
			results.Print()
			if results.Err != nil {
				return 1
			}
		}
	}

	// Explicit --file inputs are required and retain their normal command behavior.
	for _, file := range *fileCmd {
		if err := executeCommandFile(file, parser, cmdEnv, false); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	// Run interactive mode if no commands given, or if forced
	if *forceInteractive || (*executeCmd == nil && *fileCmd == nil) {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, cmdEnv, *forceTextPrompt)
		p.Run()
	}
	return 0
}
