package interactive

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/koinos/koinos-cli-wallet/internal/wallet"
)

// KoinosPrompt is an object to manage interactive mode
type KoinosPrompt struct {
	parser             *wallet.CommandParser
	execEnv            *wallet.ExecutionEnvironment
	gPrompt            *prompt.Prompt
	commandSuggestions []prompt.Suggest
}

// NewKoinosPrompt creates a new interactive prompt object
func NewKoinosPrompt(parser *wallet.CommandParser, execEnv *wallet.ExecutionEnvironment) *KoinosPrompt {
	kp := &KoinosPrompt{parser: parser, execEnv: execEnv}
	kp.gPrompt = prompt.New(kp.executor, kp.completer, prompt.OptionLivePrefix(kp.changeLivePrefix))

	// Generate command suggestions
	kp.commandSuggestions = make([]prompt.Suggest, 0)
	for _, cmd := range parser.Commands {
		if cmd.Hidden {
			continue
		}

		kp.commandSuggestions = append(kp.commandSuggestions, prompt.Suggest{Text: cmd.Name, Description: cmd.Description})
	}

	return kp
}

func (kp *KoinosPrompt) changeLivePrefix() (string, bool) {
	// Calculate online status
	onlineStatus := "🔴"
	if kp.execEnv.IsOnline() {
		onlineStatus = "🟢"
	}

	// Calculate wallet status
	walletStatus := "🔐"
	if kp.execEnv.IsWalletOpen() {
		walletStatus = "🔓"
	}

	return fmt.Sprintf("%s %s > ", onlineStatus, walletStatus), true
}

func (kp *KoinosPrompt) completer(d prompt.Document) []prompt.Suggest {
	invs, _ := kp.parser.Parse(d.Text)
	metrics := invs.Metrics()

	if metrics.CurrentParamType == wallet.CmdName {
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
	kp.gPrompt.Run()
}
