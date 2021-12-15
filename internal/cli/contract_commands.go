package cli

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/util"
	"github.com/koinos/koinos-proto-golang/encoding/text"
	"github.com/koinos/koinos-proto-golang/koinos"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
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
	ABIFilename *string
}

// NewRegisterCommand creates a new close object
func NewRegisterCommand(inv *CommandParseResult) Command {
	return &RegisterCommand{Name: *inv.Args["name"], Address: *inv.Args["address"], ABIFilename: inv.Args["abi-filename"]}
}

// Execute closes the wallet
func (c *RegisterCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if ee.Contracts.Contains(c.Name) {
		return nil, fmt.Errorf("%w: contract %s already exists", util.ErrContract, c.Name)
	}

	// Get the ABI
	var abiBytes []byte
	if c.ABIFilename != nil { // If an ABI file was given, use it
		jsonFile, err := os.Open(*c.ABIFilename)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
		}

		defer jsonFile.Close()

		abiBytes, err = ioutil.ReadAll(jsonFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
		}
	} else { // Otherwise ask the RPC server for the ABI
		meta, err := ee.RPCClient.GetContractMeta(base58.Decode(c.Address))
		if err != nil {
			return nil, err
		}

		abiBytes = []byte(meta.GetAbi())
	}

	var abi ABI
	err := json.Unmarshal(abiBytes, &abi)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
	}

	fileMap := make(map[string]*descriptorpb.FileDescriptorProto)

	// Add FieldOptions to protoregistry
	fieldProtoFile := protodesc.ToFileDescriptorProto((&descriptorpb.FieldOptions{}).ProtoReflect().Descriptor().ParentFile())
	fileMap[*fieldProtoFile.Name] = fieldProtoFile

	optionsFile := protodesc.ToFileDescriptorProto((koinos.BytesType(0)).Descriptor().ParentFile())
	fileMap[*optionsFile.Name] = optionsFile

	commonFile := protodesc.ToFileDescriptorProto((&koinos.BlockTopology{}).ProtoReflect().Descriptor().ParentFile())
	fileMap[*commonFile.Name] = commonFile

	protocolFile := protodesc.ToFileDescriptorProto((&protocol.Block{}).ProtoReflect().Descriptor().ParentFile())
	fileMap[*protocolFile.Name] = protocolFile

	chainFile := protodesc.ToFileDescriptorProto((&koinos.BlockTopology{}).ProtoReflect().Descriptor().ParentFile())
	fileMap[*chainFile.Name] = chainFile

	var fds descriptorpb.FileDescriptorSet
	err = proto.Unmarshal(abi.Types, &fds)
	if err != nil {
		fdProto := &descriptorpb.FileDescriptorProto{}
		err := proto.Unmarshal(abi.Types, fdProto)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", util.ErrInvalidABI, err)
		}

		fileMap[*fdProto.Name] = fdProto
	} else {
		for _, fdProto := range fds.GetFile() {
			fileMap[*fdProto.Name] = fdProto
		}
	}

	var protoFileOpts protodesc.FileOptions
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}

	for _, v := range fileMap {
		fileDescriptorSet.File = append(fileDescriptorSet.File, v)
	}

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

	l := md.Fields().Len()
	for i := 0; i < l; i++ {
		fd := md.Fields().Get(i)
		value := dMsg.Get(fd)

		switch fd.Kind() {
		case protoreflect.BytesKind:
			b := []byte{}
			var err error

			opts := fd.Options()
			if opts != nil {
				fieldOpts := opts.(*descriptorpb.FieldOptions)
				ext := koinos.E_Btype.TypeDescriptor()
				enum := fieldOpts.ProtoReflect().Get(ext).Enum()

				switch koinos.BytesType(enum) {
				case koinos.BytesType_HEX, koinos.BytesType_BLOCK_ID, koinos.BytesType_TRANSACTION_ID:
					b = []byte(hex.EncodeToString(value.Bytes()))
					if len(b) == 0 && len(value.Bytes()) != 0 {
						err = fmt.Errorf("error encoding hex")
					}
				case koinos.BytesType_BASE58, koinos.BytesType_CONTRACT_ID, koinos.BytesType_ADDRESS:
					b = []byte(base58.Encode(value.Bytes()))
					if len(b) == 0 && len(value.Bytes()) != 0 {
						err = fmt.Errorf("error encoding base58")
					}
				case koinos.BytesType_BASE64:
					fallthrough
				default:
					b = []byte(base64.URLEncoding.EncodeToString(value.Bytes()))
					if len(b) == 0 && len(value.Bytes()) != 0 {
						err = fmt.Errorf("error encoding base64")
					}
				}
			} else {
				b = []byte(base64.URLEncoding.EncodeToString(value.Bytes()))
				if len(b) == 0 && len(value.Bytes()) != 0 {
					err = fmt.Errorf("error encoding base64")
				}
			}

			if err != nil {
				return nil, err
			}

			value = protoreflect.ValueOfBytes(b)
		}

		// Set the value on the message
		dMsg.Set(fd, value)
	}

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
func NewWriteContractCommand(inv *CommandParseResult) Command {
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

	textMsg, _ := text.Marshal(msg)

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Calling %s with arguments '%s'", c.ParseResult.CommandName, textMsg))

	logMessage := fmt.Sprintf("Call %s with arguments '%s'", c.ParseResult.CommandName, textMsg)

	err = ee.Session.AddOperation(op, logMessage)
	if err == nil {
		er.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		id, err := ee.RPCClient.SubmitTransaction([]*protocol.Operation{op}, ee.Key)
		if err != nil {
			return nil, err
		}
		er.AddMessage(fmt.Sprintf("Submitted transaction with id %s", hex.EncodeToString(id)))
	}

	return er, nil
}
