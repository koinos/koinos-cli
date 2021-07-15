package wallet

import (
	"context"
	"fmt"
	"os"

	types "github.com/koinos/koinos-types-golang"
)

// Hardcoded Koin contract constants
const (
	ReadContractCall = "chain.read_contract"
	KoinSymbol       = "tKOIN"
	KoinPrecision    = 8
)

// CLICommand is the interface that all commands must implement
type CLICommand interface {
	Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error)
}

// ExecutionResult is the result of a command execution
type ExecutionResult struct {
	Message string
}

// NewExecutionResult creates a new execution result object
func NewExecutionResult(message string) *ExecutionResult {
	return &ExecutionResult{Message: message}
}

// ExecutionEnvironment is a struct that holds the environment for command execution.
type ExecutionEnvironment struct {
	RPCClient          *KoinosRPCClient
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

// NewCommandDeclaration create a new command declaration
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

// NewCommandArg creates a new command argument
func NewCommandArg(name string, argType CommandArgType) *CommandArg {
	return &CommandArg{
		Name:    name,
		ArgType: argType,
	}
}

// CommandArgType is an enum that defines the types of arguments a command can take
type CommandArgType int

// Types of arguments
const (
	Address = iota
	String
)

// ----------------------------------------------------------------------------
// Command Declarations
// ----------------------------------------------------------------------------

// All commands should be declared here

// BuildCommands constructs the declarations needed by the parser
func BuildCommands() []*CommandDeclaration {
	var decls []*CommandDeclaration
	decls = append(decls, NewCommandDeclaration("balance", "Check the balance at an address", false, NewBalanceCommand, *NewCommandArg("address", Address)))
	decls = append(decls, NewCommandDeclaration("create", "Create a new wallet", false, NewCreateCommand,
		*NewCommandArg("filename", String), *NewCommandArg("password", String)))
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

// BalanceCommand is a command that checks the balance of an address
type BalanceCommand struct {
	Address *types.AccountType
}

// NewBalanceCommand creates a new balance object
func NewBalanceCommand(inv *ParseResult) CLICommand {
	addressString := inv.Args["address"]
	address := types.AccountType(addressString)
	return &BalanceCommand{Address: &address}
}

// Execute fetches the balance
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
	var cResp types.ReadContractResponse
	err := ee.RPCClient.Call(ReadContractCall, params, &cResp)
	if err != nil {
		return nil, err
	}

	_, balance, err := types.DeserializeUInt64(&cResp.Result)
	if err != nil {
		return nil, err
	}

	// Build the result
	dec, err := SatoshiToDecimal(int64(*balance), KoinPrecision)
	if err != nil {
		return nil, err
	}
	er := ExecutionResult{Message: fmt.Sprintf("%v %s", dec, KoinSymbol)}

	return &er, nil
}

// ----------------------------------------------------------------------------
// Exit Command
// ----------------------------------------------------------------------------

// ExitCommand is a command that exits the wallet
type ExitCommand struct {
}

// NewExitCommand creates a new exit object
func NewExitCommand(inv *ParseResult) CLICommand {
	return &ExitCommand{}
}

// Execute exits the wallet
func (c *ExitCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	os.Exit(0)
	return nil, nil
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

// CreateCommand is a command that creates a new wallet
type CreateCommand struct {
	Filename string
	Password string
}

// NewCreateCommand creates a new create object
func NewCreateCommand(inv *ParseResult) CLICommand {
	return &CreateCommand{Filename: inv.Args["filename"], Password: inv.Args["password"]}
}

// Execute creates a new wallet
func (c *CreateCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult(fmt.Sprintf("Created wallet %s with password %s", c.Filename, c.Password))
	return result, nil
}
