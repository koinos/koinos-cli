package cli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
	"github.com/koinos/koinos-proto-golang/koinos"
	"github.com/koinos/koinos-proto-golang/koinos/chain"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	util "github.com/koinos/koinos-util-golang"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ABI is the ABI of the contract
type ABI struct {
	Methods map[string]*ABIMethod
	Types   []byte
}

// GetMethod returns the ABI method with the given name
func (abi *ABI) GetMethod(name string) *ABIMethod {
	if method, ok := abi.Methods[name]; ok {
		return method
	}

	return nil
}

// GetFiles returns the proto files of the contract
func (abi *ABI) GetFiles() (*protoregistry.Files, error) {
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

	chainFile := protodesc.ToFileDescriptorProto((&chain.DatabaseKey{}).ProtoReflect().Descriptor().ParentFile())
	fileMap[*chainFile.Name] = chainFile

	var fds descriptorpb.FileDescriptorSet
	err := proto.Unmarshal(abi.Types, &fds)
	if err != nil {
		fdProto := &descriptorpb.FileDescriptorProto{}
		err := proto.Unmarshal(abi.Types, fdProto)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
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

	return protoFileOpts.NewFiles(fileDescriptorSet)
}

// ABIMethod represents an ABI method descriptor
type ABIMethod struct {
	Argument    string `json:"argument"`
	Return      string `json:"return"`
	EntryPoint  string `json:"entry_point"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read-only"`
}

// ContractInfo represents the information about a contract
type ContractInfo struct {
	Name     string
	Address  string // []byte?
	ABI      *ABI
	Registry *protoregistry.Files
}

// Contracts is a map of contract names to ContractInfo
type Contracts map[string]*ContractInfo

// GetFromMethodName returns contract info from method name
func (c Contracts) GetFromMethodName(methodName string) *ContractInfo {
	s := strings.Split(methodName, ".")
	if len(s) != 2 {
		return nil
	}

	if !c.Contains(s[0]) {
		return nil
	}

	return c[s[0]]
}

// GetMethod returns the ABI method with the given name
func (c Contracts) GetMethod(methodName string) *ABIMethod {
	s := strings.Split(methodName, ".")
	if len(s) != 2 {
		return nil
	}

	if !c.Contains(s[0]) {
		return nil
	}

	contract := c[s[0]]

	if contract.ABI.GetMethod(s[1]) == nil {
		return nil
	}

	return contract.ABI.GetMethod(s[1])
}

// GetMethodArguments returns the message descriptor of the method arguments
func (c Contracts) GetMethodArguments(methodName string) (protoreflect.MessageDescriptor, error) {
	return c.getMethodData(methodName, true)
}

// GetMethodReturn returns the message descriptor of the method return
func (c Contracts) GetMethodReturn(methodName string) (protoreflect.MessageDescriptor, error) {
	return c.getMethodData(methodName, false)
}

func (c Contracts) getMethodData(methodName string, getArguments bool) (protoreflect.MessageDescriptor, error) {
	s := strings.Split(methodName, ".")
	if len(s) != 2 {
		return nil, fmt.Errorf("invalid method name: %s", methodName)
	}

	if !c.Contains(s[0]) {
		return nil, fmt.Errorf("contract %s does not exist", s[0])
	}

	contract := c[s[0]]
	method := c.GetMethod(methodName)

	var name string
	if getArguments {
		name = method.Argument
	} else {
		name = method.Return
	}

	// This was checked when parsing the ABI, so we should have the error. Panicking because it is really bad
	d, err := contract.Registry.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		panic(err)
	}

	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		panic(name + " is not a message")
	}

	return md, nil
}

// Contains returns true if the contract exists
func (c Contracts) Contains(name string) bool {
	_, ok := c[name]
	return ok
}

// Add adds a new contract
func (c Contracts) Add(name string, address string, abi *ABI, files *protoregistry.Files) error {
	if c.Contains(name) {
		return fmt.Errorf("contract %s already exists", name)
	}

	c[name] = &ContractInfo{
		Name:     name,
		ABI:      abi,
		Address:  address,
		Registry: files,
	}

	return nil
}

// ParseABIFields takes a message decriptor and returns a slice of command arguments
func ParseABIFields(md protoreflect.MessageDescriptor) ([]CommandArg, error) {
	return parseABIFields(md, "")
}

// ParseABIFields takes a message decriptor and returns a slice of command arguments
func parseABIFields(md protoreflect.MessageDescriptor, root string) ([]CommandArg, error) {
	params := make([]CommandArg, 0)
	l := md.Fields().Len()
	for i := 0; i < l; i++ {
		fd := md.Fields().Get(i)
		name := string(fd.Name())
		if root != "" {
			name = root + "." + name
		}

		// Translate protobuf type to parser argument type
		var t CommandArgType
		switch fd.Kind() {
		case protoreflect.BoolKind:
			t = BoolArg

		case protoreflect.Int32Kind, protoreflect.Int64Kind:
			t = IntArg

		case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			t = UIntArg

		case protoreflect.StringKind:
			t = StringArg

		case protoreflect.BytesKind:
			t = BytesArg

			opts := fd.Options()
			if opts != nil {
				fieldOpts := opts.(*descriptorpb.FieldOptions)
				ext := koinos.E_Btype.TypeDescriptor()
				enum := fieldOpts.ProtoReflect().Get(ext).Enum()

				switch koinos.BytesType(enum) {
				case koinos.BytesType_HEX, koinos.BytesType_BLOCK_ID, koinos.BytesType_TRANSACTION_ID:
					t = HexArg
				case koinos.BytesType_BASE58, koinos.BytesType_CONTRACT_ID, koinos.BytesType_ADDRESS:
					t = AddressArg
				}
			}

		case protoreflect.MessageKind:
			cmds, err := parseABIFields(fd.Message(), name)
			if err != nil {
				return nil, err
			}
			params = append(params, cmds...)
			continue

		default:
			return nil, fmt.Errorf("%w: %s", cliutil.ErrUnsupportedType, fd.Kind().String())
		}

		params = append(params, *NewCommandArg(name, t))
	}

	return params, nil
}

// DataToMessage takes a map of parsed command data and a message descriptor, and returns a message
func DataToMessage(data map[string]*string, md protoreflect.MessageDescriptor) (proto.Message, error) {
	return dataToMessage(data, md, "")
}

func dataToMessage(data map[string]*string, md protoreflect.MessageDescriptor, root string) (proto.Message, error) {
	msg := dynamicpb.NewMessage(md)
	l := md.Fields().Len()
	for i := 0; i < l; i++ {
		fd := md.Fields().Get(i)
		name := string(fd.Name())
		if root != "" {
			name = root + "." + name
		}

		inputValue := ""
		if fd.Kind() != protoreflect.MessageKind {
			inputValue = *data[name]
		}

		var value protoreflect.Value
		switch fd.Kind() {
		case protoreflect.BoolKind:
			if inputValue == "true" {
				value = protoreflect.ValueOfBool(true)
			} else {
				value = protoreflect.ValueOfBool(false)
			}

		case protoreflect.Int32Kind:
			iv, err := strconv.Atoi(inputValue)
			if err != nil {
				return nil, err
			}
			value = protoreflect.ValueOfInt32(int32(iv))

		case protoreflect.Int64Kind:
			iv, err := strconv.Atoi(inputValue)
			if err != nil {
				return nil, err
			}
			value = protoreflect.ValueOfInt64(int64(iv))

		case protoreflect.Uint32Kind:
			iv, err := strconv.Atoi(inputValue)
			if err != nil {
				return nil, err
			}
			value = protoreflect.ValueOfUint32(uint32(iv))

		case protoreflect.Uint64Kind:
			iv, err := strconv.Atoi(inputValue)
			if err != nil {
				return nil, err
			}
			value = protoreflect.ValueOfUint64(uint64(iv))

		case protoreflect.StringKind:
			value = protoreflect.ValueOfString(inputValue)

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
					b, err = util.HexStringToBytes(inputValue)
				case koinos.BytesType_BASE58, koinos.BytesType_CONTRACT_ID, koinos.BytesType_ADDRESS:
					b = base58.Decode(inputValue)
					if len(b) == 0 && len(inputValue) != 0 {
						err = errors.New("error decoding base58")
					}
				case koinos.BytesType_BASE64:
					fallthrough
				default:
					b, err = base64.URLEncoding.DecodeString(inputValue)
				}
			} else {
				b, err = base64.URLEncoding.DecodeString(inputValue)
			}

			if err != nil {
				return nil, err
			}

			value = protoreflect.ValueOfBytes(b)

		case protoreflect.MessageKind:
			subMsg, err := dataToMessage(data, fd.Message(), name)
			if err != nil {
				return nil, err
			}
			value = protoreflect.ValueOf(subMsg)

		default:
			return nil, fmt.Errorf("%w: %s", cliutil.ErrUnsupportedType, fd.Kind().String())
		}

		// Set the value on the message
		msg.Set(fd, value)
	}

	return msg, nil
}

// ParseResultToMessage takes a ParseResult and a message descriptor, and returns a message
func ParseResultToMessage(cmd *CommandParseResult, contracts Contracts) (proto.Message, error) {
	md, err := contracts.GetMethodArguments(cmd.CommandName)
	if err != nil {
		return nil, err
	}

	return DataToMessage(cmd.Args, md)
}
