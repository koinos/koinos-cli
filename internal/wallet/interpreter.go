package wallet

import (
	"context"
	"fmt"

	types "github.com/koinos/koinos-types-golang"
)

// Command execution code
// Actual command implementations are in commands.go

// CLICommand is the interface that all commands must implement
type CLICommand interface {
	Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error)
}

// ExecutionResult is the result of a command execution
type ExecutionResult struct {
	Message []string
}

// NewExecutionResult creates a new execution result object
func NewExecutionResult() *ExecutionResult {
	m := make([]string, 0)
	return &ExecutionResult{Message: m}
}

// AddMessage adds a message to the execution result
func (er *ExecutionResult) AddMessage(m string) {
	er.Message = append(er.Message, m)
}

// Print prints each message in the execution result
func (er *ExecutionResult) Print() {
	for _, m := range er.Message {
		fmt.Println(m)
	}
}

// ExecutionEnvironment is a struct that holds the environment for command execution.
type ExecutionEnvironment struct {
	RPCClient          *KoinosRPCClient
	KoinContractID     *types.ContractIDType
	KoinBalanceOfEntry types.UInt32
	Key                *KoinosKey
}

// IsWalletOpen returns a bool representing whether or not there is an open wallet
func (ee *ExecutionEnvironment) IsWalletOpen() bool {
	return ee.Key != nil
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
	Address CommandArgType = iota
	String
)

// InterpretResults is a struct that holds the results of a multi-command interpretation
type InterpretResults struct {
	Results []string
}

// NewInterpretResults creates a new InterpretResults object
func NewInterpretResults() *InterpretResults {
	ir := &InterpretResults{}
	ir.Results = make([]string, 0)
	return ir
}

// AddResult adds a result to the InterpretResults
func (ir *InterpretResults) AddResult(result ...string) {
	ir.Results = append(ir.Results, result...)
}

// Print prints the results of a command interpretation
func (ir *InterpretResults) Print() {
	for _, result := range ir.Results {
		fmt.Println(result)
	}
}

// InterpretCommands fully parses and executes a string of commands
func InterpretCommands(parser *CommandParser, ee *ExecutionEnvironment, input string) *InterpretResults {
	output := NewInterpretResults()

	invs, err := parser.Parse(input)
	if err != nil {
		output.AddResult(err.Error())
	} else {
		for _, inv := range invs {
			cmd := inv.Instantiate()
			result, err := cmd.Execute(context.Background(), ee)
			if err != nil {
				output.AddResult(err.Error())
			} else {
				output.AddResult(result.Message...)
			}
		}
	}

	return output
}
