package internal

import (
	"context"

	"github.com/ybbus/jsonrpc/v2"
)

type CLICommand interface {
	Execute(ctx context.Context, ee ExecutionEnvironment) (ExecutionResult, error)
}

type ExecutionResult struct {
	Message string
}

// ExecutionEnvironment is a struct that holds the environment for command execution.
type ExecutionEnvironment struct {
	RPCClient *jsonrpc.RPCClient
}

// ----------------------------------------------------------------------------
// Balance
// ----------------------------------------------------------------------------

type BalanceCommand struct {
}

func (c *BalanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (ExecutionResult, error) {
	return ExecutionResult{Message: "Balance"}, nil
}
