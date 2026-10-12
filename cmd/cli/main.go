package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// Other constants
const (
	rcFileName = ".koinosrc"
)

func main() {
	os.Exit(run(os.Args[1:], util.GetHomeDir()))
}

func run(args []string, homeDir string) int {
	flags := flag.NewFlagSet("koinos-cli", flag.ContinueOnError)
	// Optionally load .env file
	_ = godotenv.Load()

	// Setup command line options
	rpcAddress := flags.StringP(rpcOption, "r", rpcDefault, "RPC server URL")
	executeCmd := flags.StringSliceP(executeOption, "x", nil, "Command to execute")
	fileCmd := flags.StringSliceP(fileOption, "f", nil, "File to execute")
	versionCmd := flags.BoolP(versionOption, "v", false, "Display the version")
	forceInteractive := flags.BoolP(forceInteractiveOption, "i", false, "Forces interactive mode after successfully executing commands and files")
	forceTextPrompt := flags.BoolP(forceTextPromptOption, "t", false, "Forces text prompt in interactive mode, rather than unicode symbols")
	noRC := flags.Bool("no-rc", false, "Skip automatic home and current-directory .koinosrc files")

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
			if results.Err != nil {
				return 1
			}
		}
	}

	// Automatic startup files are optional. Explicit --file inputs are required.
	files := []string{}
	if !*noRC {
		files = append(files, filepath.Join(homeDir, rcFileName), rcFileName)
	}
	optionalFiles := len(files)
	var seenStartupFiles []os.FileInfo

	if *fileCmd != nil {
		files = append(files, *fileCmd...)
	}

	for index, file := range files {
		if index < optionalFiles {
			info, err := os.Stat(file)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				fmt.Println(err)
				return 1
			}
			duplicate := false
			for _, previous := range seenStartupFiles {
				if os.SameFile(info, previous) {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			seenStartupFiles = append(seenStartupFiles, info)
		}

		data, err := os.ReadFile(file)
		if index < optionalFiles && os.IsNotExist(err) {
			continue
		}
		if err != nil {
			fmt.Println(err)
			return 1
		}

		lines := strings.Split(string(data), "\n")
		for lineNumber, line := range lines {
			ir := cli.ParseAndInterpret(parser, cmdEnv, line)
			for _, result := range ir.Results {
				fmt.Println(result)
			}
			if ir.Err != nil {
				fmt.Printf("Stopped at %s:%d\n", file, lineNumber+1)
				return 1
			}
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
