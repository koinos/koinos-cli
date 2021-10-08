package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
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
	ee.Contracts.Add(c.Name, &abi)

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
		cmd := NewCommandDeclaration(commandName, method.EntryPoint, false, NewListCommand, params...)

		ee.Parser.Commands.AddCommand(cmd)
	}

	return &ExecutionResult{}, nil
}
