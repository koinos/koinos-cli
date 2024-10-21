package cli

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
	"github.com/koinos/koinos-proto-golang/encoding/text"
	"github.com/koinos/koinos-proto-golang/koinos"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"google.golang.org/protobuf/proto"
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
	ABIFilename *string
}

// NewRegisterCommand creates a new close object
func NewRegisterCommand(inv *CommandParseResult) Command {
	return &RegisterCommand{Name: *inv.Args["name"], Address: *inv.Args["address"], ABIFilename: inv.Args["abi-filename"]}
}

// Execute closes the wallet
func (c *RegisterCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if ee.Contracts.Contains(c.Name) {
		return nil, fmt.Errorf("%w: contract %s already exists", cliutil.ErrContract, c.Name)
	}

	// Ensure that the name is a valid command name
	_, err := ee.Parser.parseCommandName([]byte(c.Name))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid characters in contract name %s", cliutil.ErrContract, err)
	}

	// Get the ABI
	var abiBytes []byte
	if c.ABIFilename != nil { // If an ABI file was given, use it
		jsonFile, err := os.Open(*c.ABIFilename)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
		}

		defer jsonFile.Close()

		abiBytes, err = io.ReadAll(jsonFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
		}
	} else { // Otherwise ask the RPC server for the ABI
		if !ee.IsOnline() {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrOffline, "could not fetch contract ABI")
		}
		meta, err := ee.RPCClient.GetContractMeta(ctx, base58.Decode(c.Address))
		if err != nil {
			return nil, err
		}

		abiBytes = []byte(meta.GetAbi())
	}

	var abi ABI
	err = json.Unmarshal(abiBytes, &abi)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
	}

	files, err := abi.GetFiles()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
	}

	commands := []*CommandDeclaration{}

	// Iterate through the methods and construct the commands
	for name, method := range abi.Methods {
		if len(method.Argument) == 0 {
			method.Argument = "koinos.chain.nop_arguments"
		}
		d, err := files.FindDescriptorByName(protoreflect.FullName(method.Argument))
		if err != nil {
			return nil, fmt.Errorf("%w: could not find type %s", cliutil.ErrInvalidABI, method.Argument)
		}

		md, ok := d.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not a message", cliutil.ErrInvalidABI, method.Argument)
		}

		params, err := ParseABIFields(md)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
		}

		if len(method.Return) == 0 {
			method.Return = "koinos.chain.nop_result"
		}
		d, err = files.FindDescriptorByName(protoreflect.FullName(method.Return))
		if err != nil {
			return nil, fmt.Errorf("%w: could not find type %s", cliutil.ErrInvalidABI, method.Return)
		}

		_, ok = d.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not a message", cliutil.ErrInvalidABI, method.Return)
		}

		commandName := fmt.Sprintf("%s.%s", c.Name, name)

		// Create the command
		var cmd *CommandDeclaration
		if method.ReadOnly || method.ReadOnlyOld {
			cmd = NewCommandDeclaration(commandName, method.Description, false, NewReadContractCommand, params...)
		} else {
			cmd = NewCommandDeclaration(commandName, method.Description, false, NewWriteContractCommand, params...)
		}

		commands = append(commands, cmd)
	}

	// Register the contract
	err = ee.Contracts.Add(c.Name, c.Address, &abi, files)
	if err != nil {
		return nil, err
	}

	for _, cmd := range commands {
		ee.Parser.Commands.AddCommand(cmd)
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Contract '%s' at address %s registered", c.Name, c.Address))
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
func NewReadContractCommand(inv *CommandParseResult) Command {
	return &ReadContractCommand{ParseResult: inv}
}

// Execute executes the read contract command
func (c *ReadContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot execute method", cliutil.ErrOffline)
	}

	contract := ee.Contracts.GetFromMethodName(c.ParseResult.CommandName)

	entryPoint := uint64(0)

	if abiEntryPoint := ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPoint; abiEntryPoint != 0 {
		entryPoint = abiEntryPoint
	} else if abiEntryPoint := ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPointOld; len(abiEntryPoint) > 0 {
		if intEntryPoint, err := strconv.ParseUint(abiEntryPoint[2:], 16, 32); err != nil {
			return nil, err
		} else {
			entryPoint = intEntryPoint
		}
	} else {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, "method missing entry point")
	}

	// Form a protobuf message from the command input
	msg, err := ParseResultToMessage(c.ParseResult, ee.Contracts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
	}

	// Get the bytes of the message
	argBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Get the contractID
	contractID := base58.Decode(contract.Address)

	cResp, err := ee.RPCClient.ReadContract(ctx, argBytes, contractID, uint32(entryPoint))
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

	err = DecodeMessageBytes(dMsg, md)
	if err != nil {
		return nil, err
	}

	b, err := text.MarshalPretty(dMsg)
	if err != nil {
		return nil, err
	}

	er.AddMessage(string(b))

	return er, nil
}

func DecodeMessageBytes(dMsg *dynamicpb.Message, md protoreflect.MessageDescriptor) error {
	l := md.Fields().Len()
	for i := 0; i < l; i++ {
		modified := false
		fd := md.Fields().Get(i)
		value := dMsg.Get(fd)

		switch fd.Kind() {
		case protoreflect.BytesKind:
			var b []byte
			var err error

			opts := fd.Options()
			if opts != nil {
				fieldOpts := opts.(*descriptorpb.FieldOptions)
				ext := koinos.E_Btype.TypeDescriptor()
				enum := fieldOpts.ProtoReflect().Get(ext).Enum()

				switch koinos.BytesType(enum) {
				case koinos.BytesType_HEX, koinos.BytesType_BLOCK_ID, koinos.BytesType_TRANSACTION_ID:
					b, err = hex.DecodeString(string(value.Bytes()))
					if err != nil {
						return err
					}
				case koinos.BytesType_BASE58, koinos.BytesType_CONTRACT_ID, koinos.BytesType_ADDRESS:
					b = base58.Decode(string(value.Bytes()))

				case koinos.BytesType_BASE64:
					fallthrough
				default:
					b, err = base64.URLEncoding.DecodeString(string(value.Bytes()))
					if err != nil {
						return err
					}
				}
			} else {
				b, err = base64.URLEncoding.DecodeString(string(value.Bytes()))
				if err != nil {
					return err
				}
			}

			value = protoreflect.ValueOfBytes(b)
			modified = true
		}

		if fd.IsList() && value.List().Len() == 0 {
			continue
		}

		// Set the value on the message
		if modified {
			dMsg.Set(fd, value)
		}
	}

	return nil
}

// ----------------------------------------------------------------------------
// Write Contract Command
// ----------------------------------------------------------------------------

// WriteContractCommand is a backend for generated commands that write to a contract
type WriteContractCommand struct {
	ParseResult *CommandParseResult
}

// NewWriteContractCommand creates a new write contract command
func NewWriteContractCommand(inv *CommandParseResult) Command {
	return &WriteContractCommand{ParseResult: inv}
}

// Execute executes the write contract command
func (c *WriteContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot execute method", cliutil.ErrWalletClosed)
	}

	if !ee.IsOnline() && !ee.Session.IsValid() {
		return nil, fmt.Errorf("%w: cannot execute method", cliutil.ErrOffline)
	}

	contract := ee.Contracts.GetFromMethodName(c.ParseResult.CommandName)

	entryPoint := uint64(0)

	if abiEntryPoint := ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPoint; abiEntryPoint != 0 {
		entryPoint = abiEntryPoint
	} else if abiEntryPoint := ee.Contracts.GetMethod(c.ParseResult.CommandName).EntryPointOld; len(abiEntryPoint) > 0 {
		if intEntryPoint, err := strconv.ParseUint(abiEntryPoint[2:], 16, 32); err != nil {
			return nil, err
		} else {
			entryPoint = intEntryPoint
		}
	} else {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, "method missing entry point")
	}

	// Form a protobuf message from the command input
	msg, err := ParseResultToMessage(c.ParseResult, ee.Contracts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
	}

	// Get the contractID
	contractID := base58.Decode(contract.Address)

	args, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	op := &protocol.Operation{
		Op: &protocol.Operation_CallContract{
			CallContract: &protocol.CallContractOperation{
				ContractId: contractID,
				EntryPoint: uint32(entryPoint),
				Args:       args,
			},
		},
	}

	textMsg, _ := text.MarshalPretty(msg)

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Calling %s with arguments '%s'", c.ParseResult.CommandName, textMsg))

	logMessage := fmt.Sprintf("Call %s with arguments '%s'", c.ParseResult.CommandName, textMsg)

	err = ee.Session.AddOperation(op, logMessage)
	if err == nil {
		result.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		err := ee.SubmitTransaction(ctx, result, op)
		if err != nil {
			return result, fmt.Errorf("cannot make call, %w", err)
		}
	}

	return result, nil
}
