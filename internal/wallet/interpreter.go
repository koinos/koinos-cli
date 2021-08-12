package wallet

import (
	"context"
	"fmt"
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
func (er *ExecutionResult) AddMessage(m ...string) {
	er.Message = append(er.Message, m...)
}

// Print prints each message in the execution result
func (er *ExecutionResult) Print() {
	for _, m := range er.Message {
		fmt.Println(m)
	}
}

// ExecutionEnvironment is a struct that holds the environment for command execution.
type ExecutionEnvironment struct {
	RPCClient *KoinosRPCClient
	Key       *KoinosKey
	Parser    *CommandParser
}

// IsWalletOpen returns a bool representing whether or not there is an open wallet
func (ee *ExecutionEnvironment) IsWalletOpen() bool {
	return ee.Key != nil
}

// IsOnline returns a bool representing whether or not the wallet is online
func (ee *ExecutionEnvironment) IsOnline() bool {
	return ee.RPCClient != nil
}

// CommandDeclaration is a struct that declares a command
type CommandDeclaration struct {
	Name          string
	Description   string
	Instantiation func(*CommandParseResult) CLICommand
	Args          []CommandArg
	Hidden        bool // If true, the command is not shown in the help
}

func (d *CommandDeclaration) String() string {
	s := d.Name
	for _, arg := range d.Args {
		s += " <" + arg.Name + ">"
	}

	return s
}

// NewCommandDeclaration create a new command declaration
func NewCommandDeclaration(name string, description string, hidden bool,
	instantiation func(*CommandParseResult) CLICommand, args ...CommandArg) *CommandDeclaration {
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
	Name     string
	ArgType  CommandArgType
	Optional bool
}

// NewCommandArg creates a new command argument
func NewCommandArg(name string, argType CommandArgType) *CommandArg {
	return &CommandArg{
		Name:     name,
		ArgType:  argType,
		Optional: false,
	}
}

// NewOptioanlCommandArg creates a new optional command argument
func NewOptionalCommandArg(name string, argType CommandArgType) *CommandArg {
	return &CommandArg{
		Name:     name,
		ArgType:  argType,
		Optional: true,
	}
}

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

	// If there were results, skip a line at the end for readability
	if len(ir.Results) > 0 {
		fmt.Println("")
	}
}

// Interpret interprets and executes the results of a command parse
func (pr *ParseResults) Interpret(ee *ExecutionEnvironment) *InterpretResults {
	output := NewInterpretResults()

	for _, inv := range pr.CommandResults {
		cmd := inv.Instantiate()
		result, err := cmd.Execute(context.Background(), ee)
		if err != nil {
			output.AddResult(err.Error())
		} else {
			output.AddResult(result.Message...)
		}
	}

	return output
}

// ParseResultMetrics is a struct that holds various data about the parse results
// It is useful for interactive mode suggestions and error reporting
type ParseResultMetrics struct {
	CurrentResultIndex int
	CurrentArg         int
	CurrentParamType   CommandArgType
}

// Metrics is a function that returns a ParseResultMetrics object
func (pr *ParseResults) Metrics() *ParseResultMetrics {
	if len(pr.CommandResults) == 0 {
		return &ParseResultMetrics{CurrentResultIndex: 0, CurrentArg: -1, CurrentParamType: CmdName}
	}

	index := len(pr.CommandResults) - 1
	arg := pr.CommandResults[index].CurrentArg
	if pr.CommandResults[index].Termination == Command {
		index++
		arg = -1
	}

	// Calculated the type of param
	pType := CmdName
	if arg >= 0 {
		pType = pr.CommandResults[index].Decl.Args[arg].ArgType
	}

	return &ParseResultMetrics{CurrentResultIndex: index, CurrentArg: arg, CurrentParamType: pType}
}
