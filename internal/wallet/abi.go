package wallet

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// ABI is the ABI of the contract
type ABI struct {
	Methods []*ABIMethod
	Types   []byte
}

func (abi *ABI) GetMethod(name string) *ABIMethod {
	for _, method := range abi.Methods {
		if method.Name == name {
			return method
		}
	}

	return nil
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

type ContractInfo struct {
	Name string
	ABI  *ABI
}

type Contracts map[string]*ContractInfo

func (c Contracts) Contains(name string) bool {
	_, ok := c[name]
	return ok
}

func (c Contracts) Add(name string, abi *ABI) error {
	if c.Contains(name) {
		return fmt.Errorf("contract %s already exists", name)
	}

	c[name] = &ContractInfo{
		Name: name,
		ABI:  abi,
	}

	return nil
}

// ParseABIFields takes a message decriptor and returns a slice of command arguments
func ParseABIFields(md protoreflect.MessageDescriptor) ([]CommandArg, error) {
	params := make([]CommandArg, 0)
	l := md.Fields().Len()
	for i := 0; i < l; i++ {
		fd := md.Fields().Get(i)
		name := string(fd.Name())

		// Translate protobuf type to parser argument type
		var t CommandArgType
		switch fd.Kind() {
		case protoreflect.BoolKind:
			t = BoolArg

		case protoreflect.Int32Kind:
			fallthrough
		case protoreflect.Int64Kind:
			t = IntArg

		case protoreflect.Uint32Kind:
			fallthrough
		case protoreflect.Uint64Kind:
			t = UIntArg

		case protoreflect.StringKind:
			t = StringArg

		case protoreflect.BytesKind:
			t = BytesArg

		case protoreflect.MessageKind:
			cmds, err := ParseABIFields(fd.Message())
			if err != nil {
				return nil, err
			}
			params = append(params, cmds...)
			continue

		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, fd.Kind().String())
		}

		params = append(params, *NewCommandArg(name, t))

	}

	return params, nil
}
