package interactive

import (
	"fmt"

	"github.com/koinos/go-prompt"
	"github.com/koinos/go-prompt/completer"
	"github.com/koinos/koinos-cli/internal/cli"
	"github.com/koinos/koinos-cli/internal/cliutil"
)

// KoinosPrompt is an object to manage interactive mode
type KoinosPrompt struct {
	parser             *cli.CommandParser
	execEnv            *cli.ExecutionEnvironment
	gPrompt            *prompt.Prompt
	fPath              *completer.FilePathCompleter
	commandSuggestions []prompt.Suggest

	latestRevision int
}

// NewKoinosPrompt creates a new interactive prompt object
func NewKoinosPrompt(parser *cli.CommandParser, execEnv *cli.ExecutionEnvironment, forceText bool) *KoinosPrompt {
	kp := &KoinosPrompt{parser: parser, execEnv: execEnv, latestRevision: -1}
	kp.gPrompt = prompt.New(kp.executor, kp.completer, prompt.OptionLivePrefix(kp.changeLivePrefix), prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator))
	kp.fPath = &completer.FilePathCompleter{}

	return kp
}

func (kp *KoinosPrompt) generateSuggestions() {
	// Generate command suggestions
	kp.commandSuggestions = make([]prompt.Suggest, 0)
	list := kp.parser.Commands.List(false)
	for _, name := range list {
		cmd := kp.parser.Commands.Name2Command[name]
		if cmd.Hidden {
			continue
		}

		kp.commandSuggestions = append(kp.commandSuggestions, prompt.Suggest{Text: cmd.Name, Description: cmd.Description})
	}
}

func (kp *KoinosPrompt) changeLivePrefix() (string, bool) {
	return kp.execEnv.PromptStatus(), true
}

func (kp *KoinosPrompt) completer(d prompt.Document) []prompt.Suggest {
	invs, _ := kp.parser.Parse(d.Text)
	metrics := invs.Metrics()

	// Check if dirty
	if kp.latestRevision != kp.parser.Commands.Revision {
		kp.latestRevision = kp.parser.Commands.Revision
		kp.generateSuggestions()
	}

	if metrics.CurrentParamType == cli.CmdNameArg {
		return prompt.FilterHasPrefix(kp.commandSuggestions, d.GetWordBeforeCursor(), true)
	}

	if metrics.CurrentParamType == cli.FileArg {
		return kp.fPath.Complete(d)
	}

	return []prompt.Suggest{}
}

func (kp *KoinosPrompt) executor(input string) {
	results := cli.ParseAndInterpret(kp.parser, kp.execEnv, input)
	results.Print()
}

// Run runs interactive mode
func (kp *KoinosPrompt) Run() {
	fmt.Printf("Koinos CLI %s\n", cliutil.Version)
	fmt.Println("Type \"list\" for a list of commands, \"help <command>\" for help on a specific command.")
	fmt.Println("Type \"status\" to see the network, wallet, and pending operations.")
	kp.gPrompt.Run()
}
