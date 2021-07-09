package internal

import (
	"context"
	"fmt"

	"github.com/c-bata/go-prompt"
)

type InteractivePrompt struct {
	parser  *CommandParser
	execEnv *ExecutionEnvironment
	gPrompt *prompt.Prompt
}

func NewInteractivePrompt(parser *CommandParser, execEnv *ExecutionEnvironment) *InteractivePrompt {
	ip := &InteractivePrompt{parser: parser, execEnv: execEnv}
	ip.gPrompt = prompt.New(ip.executor, ip.completer)
	return ip
}

func (ip *InteractivePrompt) completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "users", Description: "Store the username and age"},
		{Text: "articles", Description: "Store the article text posted by user"},
		{Text: "comments", Description: "Store the text commented to articles"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
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
