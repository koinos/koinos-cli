package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	JSONABI = `{
		"methods": {
			"empty": {
				"argument": "abi_test.empty_arguments",
				"return": "abi_test.empty_result",
				"description": "Empty arguments",
				"entry_point": "0x2e1cfa82",
				"read-only": false
			},
			"simple": {
				"argument": "abi_test.simple_arguments",
				"return": "abi_test.simple_result",
				"description": "Simple arguments",
				"entry_point": "0xa7a39b72",
				"read-only": false
			},
			"nested": {
				"argument": "abi_test.nested_arguments",
				"return": "abi_test.nested_result",
				"description": "Nested arguments",
				"entry_point": "0x233562de",
				"read-only": false
			}
		},
		"types": "Cr4ECit0ZXN0X2FiaS9hc3NlbWJseS9wcm90by9jb25zdGVsbGF0aW9uLnByb3RvEghhYmlfdGVzdBoUa29pbm9zL29wdGlvbnMucHJvdG8iEQoPZW1wdHlfYXJndW1lbnRzIg4KDGVtcHR5X3Jlc3VsdCJOChBzaW1wbGVfYXJndW1lbnRzEg4KAmlkGAEgASgNUgJpZBISCgRuYW1lGAIgASgJUgRuYW1lEhYKBmFjdGl2ZRgDIAEoCFIGYWN0aXZlIg8KDXNpbXBsZV9yZXN1bHQiYgoQbmVzdGVkX2FyZ3VtZW50cxISCgRuYW1lGAEgASgJUgRuYW1lEiQKBGRhdGEYAiABKAsyEC5hYmlfdGVzdC5kYXRhX2NSBGRhdGESFAoFdmFsdWUYAyABKA1SBXZhbHVlIg8KDW5lc3RlZF9yZXN1bHQiRAoGZGF0YV9hEhQKBXZhbHVlGAEgASgNUgV2YWx1ZRISCgRuYW1lGAIgASgJUgRuYW1lEhAKA251bRgDIAEoCVIDbnVtIjQKBmRhdGFfYhIWCgZhY3RpdmUYASABKAhSBmFjdGl2ZRISCgRuYW1lGAIgASgJUgRuYW1lInIKBmRhdGFfYxISCgRuYW1lGAEgASgJUgRuYW1lEh4KAWEYAiABKAsyEC5hYmlfdGVzdC5kYXRhX2FSAWESFAoFdmFsdWUYAyABKA1SBXZhbHVlEh4KAWIYBCABKAsyEC5hYmlfdGVzdC5kYXRhX2JSAWJiBnByb3RvMw=="
	}`
)

func loadABI(t *testing.T) *ABI {
	var abi ABI
	err := json.Unmarshal([]byte(JSONABI), &abi)
	assert.NoError(t, err)
	return &abi
}

func loadContracts(t *testing.T) Contracts {
	contracts := Contracts(make(map[string]*ContractInfo))
	abi := loadABI(t)

	files, err := abi.GetFiles()
	assert.NoError(t, err)

	err = contracts.Add("abi_test", "", abi, files)
	assert.NoError(t, err)

	return contracts
}

func testMethod(t *testing.T, contracts Contracts, method string, expectedArguments []string) {
	arguments, err := contracts.GetMethodArguments(method)
	assert.NoError(t, err)
	ca, err := ParseABIFields(arguments)
	assert.NoError(t, err)
	assert.Equal(t, len(expectedArguments), len(ca))
	for i, expectedArgument := range expectedArguments {
		assert.Equal(t, expectedArgument, ca[i].Name)
	}
}

func TestABI(t *testing.T) {
	contracts := loadContracts(t)

	// Test empty arguments
	testMethod(t, contracts, "abi_test.empty", []string{})

	// Test simple arguments
	testMethod(t, contracts, "abi_test.simple", []string{"id", "name", "active"})

	// Test nested arguments
	testMethod(t, contracts, "abi_test.nested", []string{"name", "data.name", "data.a.value", "data.a.name", "data.a.num",
		"data.value", "data.b.active", "data.b.name", "value"})
}
