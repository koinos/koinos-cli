package cli

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	util "github.com/koinos/koinos-util-golang"
	"github.com/koinos/koinos-util-golang/rpc"
)

// Command execution code
// Actual command implementations are in commands.go

const (
	NonceCheckTime = time.Second * 60 * 3
)

// Command is the interface that all commands must implement
type Command interface {
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
	RPCClient *rpc.KoinosRPCClient
	Key       *util.KoinosKey
	Parser    *CommandParser
	Contracts Contracts
	Session   *TransactionSession

	currentNonce uint64
	nonceTime    time.Time
}

// NewExecutionEnvironment creates a new ExecutionEnvironment object
func NewExecutionEnvironment(rpcClient *rpc.KoinosRPCClient, parser *CommandParser) *ExecutionEnvironment {
	return &ExecutionEnvironment{
		RPCClient:    rpcClient,
		Parser:       parser,
		Contracts:    make(map[string]*ContractInfo),
		Session:      &TransactionSession{},
		nonceTime:    time.Time{},
		currentNonce: 0,
	}
}

// OpenWallet opens a wallet
func (ee *ExecutionEnvironment) OpenWallet(key *util.KoinosKey) {
	ee.Key = key
	ee.ResetNonce()
}

// CloseWallet closes the wallet
func (ee *ExecutionEnvironment) CloseWallet() {
	ee.Key = nil
	ee.ResetNonce()
}

// ResetNonce resets the nonce
func (ee *ExecutionEnvironment) ResetNonce() {
	ee.currentNonce = 0
	ee.nonceTime = time.Time{}
}

// GetNonce returns the current nonce
func (ee *ExecutionEnvironment) GetNonce() (uint64, error) {
	if ee.nonceTime.IsZero() || time.Now().Sub(ee.nonceTime) > NonceCheckTime {
		nonce, err := ee.RPCClient.GetAccountNonce(ee.Key.AddressBytes())
		if err != nil {
			return 0, err
		}

		atomic.StoreUint64(&ee.currentNonce, nonce)
	}

	ee.nonceTime = time.Now()

	atomic.AddUint64(&ee.currentNonce, 1)
	return ee.currentNonce, nil
}

// GetRcLimit returns the current RC limit
func (ee *ExecutionEnvironment) GetRcLimit() (uint64, error) {
	return ee.RPCClient.GetAccountRc(ee.Key.AddressBytes())
}

// GetSubmissionParams returns the submission parameters for a command
func (ee *ExecutionEnvironment) GetSubmissionParams() (*rpc.SubmissionParams, error) {
	nonce, err := ee.GetNonce()
	if err != nil {
		return nil, err
	}

	rcLimit, err := ee.GetRcLimit()
	if err != nil {
		return nil, err
	}

	return &rpc.SubmissionParams{
		Nonce:   nonce,
		RCLimit: rcLimit,
	}, nil
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
	Instantiation func(*CommandParseResult) Command
	Args          []CommandArg
	Hidden        bool // If true, the command is not shown in the help
}

func (d *CommandDeclaration) String() string {
	s := d.Name
	for _, arg := range d.Args {
		s += fmt.Sprintf(" %s", arg.String())
	}

	return s
}

// NewCommandDeclaration create a new command declaration
func NewCommandDeclaration(name string, description string, hidden bool,
	instantiation func(*CommandParseResult) Command, args ...CommandArg) *CommandDeclaration {
	// Ensure optionals are only at the end
	req := true
	for _, arg := range args {
		if !arg.Optional {
			if !req {
				return nil
			}
		} else {
			req = false
		}
	}

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

// NewOptionalCommandArg creates a new optional command argument
func NewOptionalCommandArg(name string, argType CommandArgType) *CommandArg {
	return &CommandArg{
		Name:     name,
		ArgType:  argType,
		Optional: true,
	}
}

func (arg *CommandArg) String() string {
	filling := fmt.Sprintf("%s:%s", arg.Name, arg.ArgType.String())
	var val string
	if arg.Optional {
		val = "[" + filling + "]"
	} else {
		val = "<" + filling + ">"
	}

	return val
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
		return &ParseResultMetrics{CurrentResultIndex: 0, CurrentArg: -1, CurrentParamType: CmdNameArg}
	}

	index := len(pr.CommandResults) - 1
	arg := pr.CommandResults[index].CurrentArg
	if pr.CommandResults[index].Termination == CommandTermination {
		index++
		arg = -1
	}

	// Calculated the type of param
	pType := CmdNameArg
	if arg >= 0 {
		// If there is a declaration, find the type of the param
		if pr.CommandResults[index].Decl != nil {
			pType = pr.CommandResults[index].Decl.Args[arg].ArgType
		} else { // Otherwise it is an invalid command
			pType = NoArg
		}
	}

	return &ParseResultMetrics{CurrentResultIndex: index, CurrentArg: arg, CurrentParamType: pType}
}

// ParseAndInterpret is a helper function to parse and interpret the given command string
func ParseAndInterpret(parser *CommandParser, ee *ExecutionEnvironment, input string) *InterpretResults {
	result, err := parser.Parse(input)
	if err != nil {
		o := NewInterpretResults()
		o.AddResult(err.Error())
		metrics := result.Metrics()
		// Display help for the command if it is a valid command
		if len(result.CommandResults) > 0 && result.CommandResults[metrics.CurrentResultIndex].Decl != nil {
			o.AddResult("Usage: " + result.CommandResults[metrics.CurrentResultIndex].Decl.String())
		} else {
			o.AddResult("Type \"list\" for a list of commands.")
		}
		return o
	}

	return result.Interpret(ee)
}
