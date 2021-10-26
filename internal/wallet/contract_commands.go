package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli-wallet/internal/util"
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
		return nil, fmt.Errorf("%w: contract %s already exists", util.ErrContract, c.Name)
	}

	jsonFile, err := os.Open(c.ABIFilename)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	defer jsonFile.Close()

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	var abi ABI
	err = json.Unmarshal(jsonBytes, &abi)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	fieldProto := descriptorpb.FieldOptions{}
	fileDescriptorSet.File = append(fileDescriptorSet.File, protodesc.ToFileDescriptorProto(fieldProto.ProtoReflect().Descriptor().ParentFile()))

	var fds descriptorpb.FileDescriptorSet
	err = proto.Unmarshal(abi.Types, &fds)
	if err != nil {
		fdProto := &descriptorpb.FileDescriptorProto{}
		err := proto.Unmarshal(abi.Types, fdProto)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
		}

		fileDescriptorSet.File = append(fileDescriptorSet.File, fdProto)
	} else {
		for _, fdProto := range fds.GetFile() {
			fileDescriptorSet.File = append(fileDescriptorSet.File, fdProto)
		}
	}

	var protoFileOpts protodesc.FileOptions
	files, err := protoFileOpts.NewFiles(fileDescriptorSet)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	commands := []*CommandDeclaration{}

	// Iterate through the methods and construct the commands
	for name, method := range abi.Methods {
		d, err := files.FindDescriptorByName(protoreflect.FullName(method.Argument))
		if err != nil {
			return nil, fmt.Errorf("%w: could not find type %s", util.ErrInvalidABI, method.Argument)
		}

		md, ok := d.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not a message", util.ErrInvalidABI, method.Argument)
		}

		params, err := ParseABIFields(md)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
		}

		d, err = files.FindDescriptorByName(protoreflect.FullName(method.Return))
		if err != nil {
			return nil, fmt.Errorf("%w: could not find type %s", util.ErrInvalidABI, method.Argument)
		}

		md, ok = d.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not a message", util.ErrInvalidABI, method.Argument)
		}

		commandName := fmt.Sprintf("%s.%s", c.Name, name)

		// Create the command
		var cmd *CommandDeclaration
		if method.ReadOnly {
			cmd = NewCommandDeclaration(commandName, method.Description, false, NewReadContractCommand, params...)
		} else {
			cmd = NewCommandDeclaration(commandName, method.Description, false, NewWriteContractCommand, params...)
		}

		commands = append(commands, cmd)
	}

	// Register the contract
	ee.Contracts.Add(c.Name, c.Address, &abi, files)

	for _, cmd := range commands {
		ee.Parser.Commands.AddCommand(cmd)
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Contract '%s' at address %s registered.", c.Name, c.Address))
	return er, nil
}

// ----------------------------------------------------------------------------
// Read Contract Command
// ----------------------------------------------------------------------------

// ReadContractCommand is a backend for generated commands that read from a contract
type ReadContractCommand struct {
	ParseResult *CommandParseResult
}

// NewReadContractCommand creates a new read contract command
func NewReadContractCommand(inv *CommandParseResult) CLICommand {
	return &ReadContractCommand{ParseResult: inv}
}

// Execute executes the read contract command
func (c *ReadContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot execute method", util.ErrOffline)
	}

	contract := ee.Contracts.GetFromMethodName(c.ParseResult.CommandName)

	entryPoint, err := strconv.ParseUint(ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	// Form a protobuf message from the command input
	msg, err := ParseResultToMessage(c.ParseResult, ee.Contracts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	// Get the bytes of the message
	argBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Get the contractID
	contractID := base58.Decode(contract.Address)

	cResp, err := ee.RPCClient.ReadContract(argBytes, contractID, uint32(entryPoint))
	if err != nil {
		return nil, err
	}

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

// WriteContractCommand is a backend for generated commands that write to a contract
type WriteContractCommand struct {
	ParseResult *CommandParseResult
}

// NewWriteContractCommand creates a new write contract command
func NewWriteContractCommand(inv *CommandParseResult) CLICommand {
	return &WriteContractCommand{ParseResult: inv}
}

// Execute executes the write contract command
func (c *WriteContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot execute method", util.ErrWalletClosed)
	}

	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot execute method", util.ErrOffline)
	}

	contract := ee.Contracts.GetFromMethodName(c.ParseResult.CommandName)

	entryPoint, err := strconv.ParseUint(ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	// Form a protobuf message from the command input
	msg, err := ParseResultToMessage(c.ParseResult, ee.Contracts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	// Get the contractID
	contractID := base58.Decode(contract.Address)

	_, err = ee.RPCClient.WriteMessageContract(msg, ee.Key, contractID, uint32(entryPoint))
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Transaction submitted to contract '%s' at address %s .", contract.Name, contract.Address))

	return er, nil
}
