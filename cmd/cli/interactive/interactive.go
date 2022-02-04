package interactive

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/koinos/koinos-cli/internal/cli"
	"github.com/koinos/koinos-cli/internal/cliutil"
)

// KoinosPrompt is an object to manage interactive mode
type KoinosPrompt struct {
	parser             *cli.CommandParser
	execEnv            *cli.ExecutionEnvironment
	gPrompt            *prompt.Prompt
	commandSuggestions []prompt.Suggest
	unicodeSupport     bool

	latestRevision int

	onlineDisplay  string
	offlineDisplay string
	openDisplay    string
	closeDisplay   string
	sessionDisplay string
}

// NewKoinosPrompt creates a new interactive prompt object
func NewKoinosPrompt(parser *cli.CommandParser, execEnv *cli.ExecutionEnvironment) *KoinosPrompt {
	kp := &KoinosPrompt{parser: parser, execEnv: execEnv, latestRevision: -1}
	kp.gPrompt = prompt.New(kp.executor, kp.completer, prompt.OptionLivePrefix(kp.changeLivePrefix))

	// Check for terminal unicode support
	lang := strings.ToUpper(os.Getenv("LANG"))
	kp.unicodeSupport = strings.Contains(lang, "UTF")

	// Setup status characters
	if kp.unicodeSupport {
		kp.onlineDisplay = ""
		kp.offlineDisplay = "🚫 "
		kp.closeDisplay = "🔐 "
		kp.openDisplay = "🔓 "
		kp.sessionDisplay = "📄 "
	} else {
		kp.onlineDisplay = "(online) "
		kp.offlineDisplay = "(offline) "
		kp.closeDisplay = "(locked) "
		kp.openDisplay = "(unlocked) "
		kp.sessionDisplay = "(session) "
	}

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
	// Calculate online status
	onlineStatus := kp.offlineDisplay
	if kp.execEnv.IsOnline() {
		onlineStatus = kp.onlineDisplay
	}

	// Calculate wallet status
	walletStatus := kp.closeDisplay
	if kp.execEnv.IsWalletOpen() {
		walletStatus = kp.openDisplay
	}

	sessionStatus := ""
	if kp.execEnv.Session.IsValid() {
		sessionStatus = kp.sessionDisplay
	}

	return fmt.Sprintf("%s%s%s> ", onlineStatus, walletStatus, sessionStatus), true
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

	return []prompt.Suggest{}
}

func (kp *KoinosPrompt) executor(input string) {
	results := cli.ParseAndInterpret(kp.parser, kp.execEnv, input)
	results.Print()
}

// Run runs interactive mode
func (kp *KoinosPrompt) Run() {
	fmt.Println(fmt.Sprintf("Koinos CLI %s", cliutil.Version))
	fmt.Println("Type \"list\" for a list of commands, \"help <command>\" for help on a specific command.")
	kp.gPrompt.Run()
}
