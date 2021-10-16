package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ----------------------------------------------------------------------------
// Register Command
// ----------------------------------------------------------------------------

// RegisterCommand is a command that closes an open wallet
type RegisterCommand struct {
	Name        string
	Address     string
	ABIFilename string
}

// NewRegisterCommand creates a new close object
func NewRegisterCommand(inv *CommandParseResult) CLICommand {
	return &RegisterCommand{Name: *inv.Args["name"], Address: *inv.Args["address"], ABIFilename: *inv.Args["abi-filename"]}
}

// Execute closes the wallet
func (c *RegisterCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if ee.Contracts.Contains(c.Name) {
		return nil, fmt.Errorf("%w: contract %s already exists", ErrContract, c.Name)
	}

	jsonFile, err := os.Open(c.ABIFilename)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
	}

	defer jsonFile.Close()

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
	}

	var abi ABI
	err = json.Unmarshal(jsonBytes, &abi)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
	}

	var fds descriptorpb.FileDescriptorSet
	err = proto.Unmarshal(abi.Types, &fds)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
	}

	var protoFileOpts protodesc.FileOptions
	files, err := protoFileOpts.NewFiles(&fds)

	if files.NumFiles() != 1 {
		return nil, fmt.Errorf("%w: expected 1 descriptor, got %d", ErrInvalidABI, files.NumFiles())
	}

	// Get the file descriptor
	var fDesc protoreflect.FileDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fDesc = fd
		return true
	})

	// Register the contract
	ee.Contracts.Add(c.Name, c.Address, &abi, fDesc)

	// Iterate through the methods and construct the commands
	for _, method := range abi.Methods {
		d := fDesc.Messages().ByName(protoreflect.Name(method.Argument))
		if d == nil {
			return nil, fmt.Errorf("%w: could not find type %s", ErrInvalidABI, method.Argument)
		}

		params, err := ParseABIFields(d)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
		}

		commandName := fmt.Sprintf("%s.%s", c.Name, method.Name)

		// Create the command
		var cmd *CommandDeclaration
		if method.ReadOnly {
			cmd = NewCommandDeclaration(commandName, method.Description, false, NewReadContractCommand, params...)
		} else {
			cmd = NewCommandDeclaration(commandName, method.Description, false, NewWriteContractCommand, params...)
		}

		ee.Parser.Commands.AddCommand(cmd)
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Contract '%s' at address %s registered.", c.Name, c.Address))
	return er, nil
}

// ----------------------------------------------------------------------------
// Read Contract Command
// ----------------------------------------------------------------------------

type ReadContractCommand struct {
	ParseResult *CommandParseResult
}

// NewRegisterCommand creates a new close object
func NewReadContractCommand(inv *CommandParseResult) CLICommand {
	return &ReadContractCommand{ParseResult: inv}
}

func (c *ReadContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	contract := ee.Contracts.GetFromMethodName(c.ParseResult.CommandName)

	entryPoint, err := strconv.ParseInt(ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	// Form a protobuf message from the command input
	msg, err := ParseResultToMessage(c.ParseResult, ee.Contracts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
	}

	// Get the bytes of the message
	argBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Get the contractID
	contractID, err := HexStringToBytes(contract.Address)
	if err != nil {
		panic("Invalid contract ID")
	}

	cResp, err := ee.RPCClient.ReadContract(argBytes, contractID, uint32(entryPoint))
	if err != nil {
		return nil, err
	}

	//balanceOfReturn := &token.BalanceOfResult{}
	//err = proto.Unmarshal(cResp.Result, balanceOfReturn)
	//if err != nil {
	//	return 0, err
	//}

	// Get return message descriptor
	md, err := ee.Contracts.GetMethodReturn(c.ParseResult.CommandName)
	if err != nil {
		return nil, err
	}

	dMsg := dynamicpb.NewMessage(md)
	err = proto.Unmarshal(cResp.GetResult(), dMsg)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()

	b, err := prototext.Marshal(dMsg)
	if err != nil {
		return nil, err
	}

	er.AddMessage(string(b))

	return er, nil
}

// ----------------------------------------------------------------------------
// Write Contract Command
// ----------------------------------------------------------------------------

type WriteContractCommand struct {
	ParseResult *CommandParseResult
}

// NewRegisterCommand creates a new close object
func NewWriteContractCommand(inv *CommandParseResult) CLICommand {
	return &WriteContractCommand{ParseResult: inv}
}

func (c *WriteContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	contract := ee.Contracts.GetFromMethodName(c.ParseResult.CommandName)

	entryPoint, err := strconv.ParseInt(ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	// Form a protobuf message from the command input
	msg, err := ParseResultToMessage(c.ParseResult, ee.Contracts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
	}

	// Get the bytes of the message
	argBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Get the contractID
	contractID, err := HexStringToBytes(contract.Address)
	if err != nil {
		panic("Invalid contract ID")
	}

	cResp, err := ee.RPCClient.ReadContract(argBytes, contractID, uint32(entryPoint))
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("%d", len(cResp.Result)))

	return er, nil
}
