package interactive

import (
	"context"
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/koinos/koinos-cli-wallet/internal/wallet"
)

type InteractivePrompt struct {
	parser             *wallet.CommandParser
	execEnv            *wallet.ExecutionEnvironment
	gPrompt            *prompt.Prompt
	commandSuggestions []prompt.Suggest
}

func NewInteractivePrompt(parser *wallet.CommandParser, execEnv *wallet.ExecutionEnvironment) *InteractivePrompt {
	ip := &InteractivePrompt{parser: parser, execEnv: execEnv}
	ip.gPrompt = prompt.New(ip.executor, ip.completer)

	// Generate command suggestions
	ip.commandSuggestions = make([]prompt.Suggest, 0)
	for _, cmd := range parser.Commands {
		if cmd.Hidden {
			continue
		}

		ip.commandSuggestions = append(ip.commandSuggestions, prompt.Suggest{Text: cmd.Name, Description: cmd.Description})
	}

	return ip
}

func (ip *InteractivePrompt) completer(d prompt.Document) []prompt.Suggest {
	var current_inv *wallet.ParseResult
	invs, _ := ip.parser.Parse(d.Text)
	if len(invs) != 0 {
		current_inv = invs[len(invs)-1]
	}

	if current_inv == nil || current_inv.CurrentArg == -1 {
		return prompt.FilterHasPrefix(ip.commandSuggestions, d.GetWordBeforeCursor(), true)
	}

	return []prompt.Suggest{}
}

func (ip *InteractivePrompt) executor(input string) {
	invs, err := ip.parser.Parse(input)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, inv := range invs {
		cmd := inv.Instantiate()
		result, err := cmd.Execute(context.Background(), ip.execEnv)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(result.Message)
		}
	}
}

func (ip *InteractivePrompt) Run() {
	ip.gPrompt.Run()
}
