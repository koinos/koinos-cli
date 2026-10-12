package interactive

import (
	"testing"

	"github.com/koinos/go-prompt"
	"github.com/koinos/koinos-cli/internal/cli"
)

func TestWalletCommandsAreRemovedFromHistoryEvenWhenTheyFail(t *testing.T) {
	for _, input := range []string{"open missing.wallet inline-secret", "list; import private-key missing.wallet inline-secret", "create", `open"missing.wallet" disposable-review-secret`, `import'WIF' wallet password`} {
		parser := cli.NewCommandParser(cli.NewKoinosCommandSet())
		history := prompt.NewHistory()
		history.Add("status")
		history.Add(input)
		kp := &KoinosPrompt{parser: parser, execEnv: cli.NewExecutionEnvironment(nil, parser), gPrompt: &prompt.Prompt{History: history}}
		kp.executor(input)
		if len(history.Histories) != 1 || history.Histories[0] != "status" {
			t.Fatal("wallet command remained in interactive history")
		}
	}
}
