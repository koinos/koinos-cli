package wallet

import (
	"context"
	"fmt"
	"os"

	types "github.com/koinos/koinos-types-golang"
	"github.com/ybbus/jsonrpc/v2"
)

const (
	ReadContractCall = "chain.read_contract"
	KoinSymbol       = "tKOIN"
)

type CLICommand interface {
	Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error)
}

type ExecutionResult struct {
	Message string
}

// ExecutionEnvironment is a struct that holds the environment for command execution.
type ExecutionEnvironment struct {
	RPCClient          jsonrpc.RPCClient
	KoinContractID     *types.ContractIDType
	KoinBalanceOfEntry types.UInt32
}

// CommandDeclaration is a struct that declares a command
type CommandDeclaration struct {
	Name          string
	Description   string
	Instantiation func(*ParseResult) CLICommand
	Args          []CommandArg
	Hidden        bool // If true, the command is not shown in the help
}

func NewCommandDeclaration(name string, description string, hidden bool,
	instantiation func(*ParseResult) CLICommand, args ...CommandArg) *CommandDeclaration {
	return &CommandDeclaration{
		Name:          name,
		Description:   description,
		Hidden:        hidden,
		Instantiation: instantiation,
		Args:          args,
	}
}

// CommandArg is a struct that holds an argument for a command
type CommandArg struct {
	Name    string
	ArgType CommandArgType
}

func NewCommandArg(name string, argType CommandArgType) *CommandArg {
	return &CommandArg{
		Name:    name,
		ArgType: argType,
	}
}

// CommandArgType is an enum that defines the types of arguments a command can take
type CommandArgType int

const (
	Address = iota
)

// ----------------------------------------------------------------------------
// Command Declarations
// ----------------------------------------------------------------------------

// All commands should be declared here

// BuildCommands constructs the declarations needed by the parser
func BuildCommands() []*CommandDeclaration {
	var decls []*CommandDeclaration
	decls = append(decls, NewCommandDeclaration("balance", "Check the balance at an address", false, NewBalanceCommand, *NewCommandArg("address", Address)))
	decls = append(decls, NewCommandDeclaration("exit", "Exit the wallet (quit also works)", false, NewExitCommand))
	decls = append(decls, NewCommandDeclaration("quit", "", true, NewExitCommand))

	return decls
}

// ----------------------------------------------------------------------------
// Command Implementations
// ----------------------------------------------------------------------------

// All commands should be implemented here

// ----------------------------------------------------------------------------
// Balance Command
// ----------------------------------------------------------------------------

type BalanceCommand struct {
	Address *types.AccountType
}

func NewBalanceCommand(inv *ParseResult) CLICommand {
	address_string := inv.Args["address"]
	address := types.AccountType(address_string)
	return &BalanceCommand{Address: &address}
}

func (c *BalanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Build the contract request
	params := types.NewReadContractRequest()
	params.ContractID = *ee.KoinContractID
	params.EntryPoint = ee.KoinBalanceOfEntry
	// Serialize the args
	vb := types.NewVariableBlob()
	vb = c.Address.Serialize(vb)
	params.Args = *vb

	// Make the rpc call
	resp, err := ee.RPCClient.Call(ReadContractCall, params)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Fetch the contract response
	var c_resp types.ReadContractResponse
	err = resp.GetObject(&c_resp)
	if err != nil {
		return nil, err
	}

	_, balance, err := types.DeserializeUInt64(&c_resp.Result)
	if err != nil {
		return nil, err
	}

	// Build the result
	er := ExecutionResult{Message: fmt.Sprintf("%v %s", KoinToDecimal(balance), KoinSymbol)}

	return &er, nil
}

// ----------------------------------------------------------------------------
// Exit Command
// ----------------------------------------------------------------------------

type ExitCommand struct {
}

func NewExitCommand(inv *ParseResult) CLICommand {
	return &ExitCommand{}
}

func (c *ExitCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	os.Exit(0)
	return nil, nil
}
