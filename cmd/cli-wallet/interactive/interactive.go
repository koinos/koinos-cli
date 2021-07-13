package interactive

import (
	"context"
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
	kp.gPrompt = prompt.New(kp.executor, kp.completer)

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

func (kp *KoinosPrompt) completer(d prompt.Document) []prompt.Suggest {
	var currentInv *wallet.ParseResult
	invs, _ := kp.parser.Parse(d.Text)
	if len(invs) != 0 {
		currentInv = invs[len(invs)-1]
	}

	if len(d.Text) == 0 || currentInv != nil && currentInv.CurrentArg == -1 {
		return prompt.FilterHasPrefix(kp.commandSuggestions, d.GetWordBeforeCursor(), true)
	}

	return []prompt.Suggest{}
}

func (kp *KoinosPrompt) executor(input string) {
	invs, err := kp.parser.Parse(input)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, inv := range invs {
		cmd := inv.Instantiate()
		result, err := cmd.Execute(context.Background(), kp.execEnv)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(result.Message)
		}
	}
}

// Run runs interactive mode
func (kp *KoinosPrompt) Run() {
	kp.gPrompt.Run()
}
