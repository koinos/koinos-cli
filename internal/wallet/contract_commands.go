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

// ABI is the ABI of the contract
type ABI struct {
	Methods []*ABIMethod
	Types   []byte
}

// ABIMethod represents an ABI method descriptor
type ABIMethod struct {
	Name        string `json:"name"`
	Argument    string `json:"argument"`
	Return      string `json:"return"`
	EntryPoint  string `json:"entry_point"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read-only"`
}

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
	//var protoFiles protoregistry.Files
	files, err := protoFileOpts.NewFiles(&fds)

	if files.NumFiles() != 1 {
		return nil, fmt.Errorf("expected 1 descriptor, got %d", files.NumFiles())
	}

	// Get the file descriptor
	var fDesc protoreflect.FileDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fDesc = fd
		return true
	})

	// Iterate through the methods and construct the commands
	for _, method := range abi.Methods {
		d := fDesc.Messages().ByName(protoreflect.Name(method.Argument))
		params, err := ParseABIFields(d)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidABI, err)
		}

		fmt.Println(params)
	}

	return &ExecutionResult{}, nil
}
