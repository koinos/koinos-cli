package interactive

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/koinos/koinos-cli-wallet/internal/wallet"
)

// KoinosPrompt is an object to manage interactive mode
type KoinosPrompt struct {
	parser             *wallet.CommandParser
	execEnv            *wallet.ExecutionEnvironment
	gPrompt            *prompt.Prompt
	commandSuggestions []prompt.Suggest
	unicodeSupport     bool

	onlineDisplay  string
	offlineDisplay string
	openDisplay    string
	closeDisplay   string
}

// NewKoinosPrompt creates a new interactive prompt object
func NewKoinosPrompt(parser *wallet.CommandParser, execEnv *wallet.ExecutionEnvironment) *KoinosPrompt {
	kp := &KoinosPrompt{parser: parser, execEnv: execEnv}
	kp.gPrompt = prompt.New(kp.executor, kp.completer, prompt.OptionLivePrefix(kp.changeLivePrefix))

	// Generate command suggestions
	kp.commandSuggestions = make([]prompt.Suggest, 0)
	list := parser.Commands.List(false)
	for _, name := range list {
		cmd := parser.Commands.Name2Command[name]
		if cmd.Hidden {
			continue
		}

		kp.commandSuggestions = append(kp.commandSuggestions, prompt.Suggest{Text: cmd.Name, Description: cmd.Description})
	}

	// Check for terminal unicode support
	lang := strings.ToUpper(os.Getenv("LANG"))
	kp.unicodeSupport = strings.Contains(lang, "UTF")

	// Setup status characters
	if kp.unicodeSupport {
		kp.onlineDisplay = "🟢"
		kp.offlineDisplay = "🔴"
		kp.closeDisplay = "🔐"
		kp.openDisplay = "🔓"
	} else {
		kp.onlineDisplay = "(online)"
		kp.offlineDisplay = "(offline)"
		kp.closeDisplay = "(locked)"
		kp.openDisplay = "(unlocked)"
	}

	return kp
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

	return fmt.Sprintf("%s %s > ", onlineStatus, walletStatus), true
}

func (kp *KoinosPrompt) completer(d prompt.Document) []prompt.Suggest {
	invs, _ := kp.parser.Parse(d.Text)
	metrics := invs.Metrics()

	if metrics.CurrentParamType == wallet.CmdNameArg {
		return prompt.FilterHasPrefix(kp.commandSuggestions, d.GetWordBeforeCursor(), true)
	}

	return []prompt.Suggest{}
}

func (kp *KoinosPrompt) executor(input string) {
	results := wallet.ParseAndInterpret(kp.parser, kp.execEnv, input)
	results.Print()
}

// Run runs interactive mode
func (kp *KoinosPrompt) Run() {
	fmt.Println(fmt.Sprintf("Koinos CLI Wallet %s", wallet.Version))
	fmt.Println("Type \"list\" for a list of commands, \"help <command>\" for help on a specific command.")
	kp.gPrompt.Run()
}
