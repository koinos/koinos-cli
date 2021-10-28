package cli

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/koinos/koinos-cli/internal/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSatoshiToDecimal(t *testing.T) {
	v, err := util.SatoshiToDecimal(100000000, 8)
	if err != nil {
		t.Error(err)
	}

	if !v.Equal(decimal.NewFromFloat(1.0)) {
		t.Error("Expected 1.0, got", v)
	}

	v, err = util.SatoshiToDecimal(1000, 1)
	if err != nil {
		t.Error(err)
	}

	if !v.Equal(decimal.NewFromFloat(100.0)) {
		t.Error("Expected 100.0, got", v)
	}

	v, err = util.SatoshiToDecimal(12345678, 3)
	if err != nil {
		t.Error(err)
	}

	if !v.Equal(decimal.NewFromFloat(12345.678)) {
		t.Error("Expected 1234.5678, got", v)
	}
}

func makeTestParser() *CommandParser {
	// Construct the command parser
	cs := NewCommandSet()

	cs.AddCommand(NewCommandDeclaration("test_address", "Test command which takes an address", false, nil, *NewCommandArg("address", AddressArg)))
	cs.AddCommand(NewCommandDeclaration("test_string", "Test command which takes a string", false, nil, *NewCommandArg("string", StringArg)))
	cs.AddCommand(NewCommandDeclaration("test_none", "Test command which takes no arguments", false, nil))
	cs.AddCommand(NewCommandDeclaration("test_none2", "Another test command which takes no arguments", false, nil))
	cs.AddCommand(NewCommandDeclaration("test_multi", "Test command which takes multiple arguments, and of different types", false, NewGenerateKeyCommand,
		*NewCommandArg("arg0", AddressArg), *NewCommandArg("arg1", StringArg), *NewCommandArg("arg2", AmountArg), *NewCommandArg("arg0", StringArg)))
	cs.AddCommand(NewCommandDeclaration("optional", "Test command which takes optional arguments", false, nil, *NewCommandArg("arg0", StringArg),
		*NewCommandArg("arg1", StringArg), *NewOptionalCommandArg("arg2", StringArg), *NewOptionalCommandArg("arg3", StringArg)))
	cs.AddCommand(NewCommandDeclaration("test_bool", "Test command which takes a boolean", false, nil, *NewCommandArg("string", StringArg),
		*NewCommandArg("bool", BoolArg), *NewCommandArg("amount", AmountArg)))

	parser := NewCommandParser(cs)

	return parser
}

func TestBasicParser(t *testing.T) {
	parser := makeTestParser()

	// Test parsing several commands
	results, err := parser.Parse("test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg; test_none; test_none2")
	if err != nil {
		t.Error(err)
	}

	if results.Len() != 3 {
		t.Error("Expected 3 result, got", results.Len())
	}

	results, err = parser.Parse("asdasd")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if !errors.Is(err, util.ErrUnknownCommand) {
		t.Error("Expected error", util.ErrUnknownCommand, ", got", err)
	}

	if results.CommandResults[0].CurrentArg != -1 {
		t.Error("Expected current arg to be -1, got", results.CommandResults[0].CurrentArg)
	}

	results, err = parser.Parse("asdasd ")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if !errors.Is(err, util.ErrUnknownCommand) {
		t.Error("Expected error", util.ErrUnknownCommand, ", got", err)
	}

	if results.CommandResults[0].CurrentArg != 0 {
		t.Error("Expected current arg to be 0, got", results.CommandResults[0].CurrentArg)
	}

	// Test parsing empty inputs
	results, err = parser.Parse("")
	if err != nil {
		t.Error(err)
	}

	if results.Len() != 0 {
		t.Error("Expected 0 results, got", results.Len())
	}

	results, err = parser.Parse("    ")
	if err != nil {
		t.Error(err)
	}

	if results.Len() != 0 {
		t.Error("Expected 0 results, got", results.Len())
	}
}

func TestBadInput(t *testing.T) {
	parser := makeTestParser()

	// Test nonsensical string of empty commands
	results, err := parser.Parse(" ; ;; ;; ;;;;    ;     ;  ;    ")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if results.Len() != 0 {
		t.Error("Expected 0 results, got", results.Len())
	}

	// Test valid command followed by empty commands
	results, err = parser.Parse("test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg;  ;;  ; ;; test_none")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if !errors.Is(err, util.ErrEmptyCommandName) {
		t.Error("Expected error", util.ErrEmptyCommandName, ", got", err)
	}

	if results.Len() != 1 {
		t.Error("Expected 1 result, got", results.Len())
	}
}

func TestOptionalArguments(t *testing.T) {
	parser := makeTestParser()

	// These should error since it is missing a required argument
	checkParseResults(t, parser, "optional", util.ErrMissingParam, []string{}, []interface{}{})
	checkParseResults(t, parser, "optional abcd", util.ErrMissingParam, []string{}, []interface{}{})

	// Check with proper optional arguments
	checkParseResults(t, parser, "optional abcd efgh", nil, []string{"arg0", "arg1", "arg2", "arg3"}, []interface{}{"abcd", "efgh", nil, nil})
	checkParseResults(t, parser, "optional abcd efgh ijkl", nil, []string{"arg0", "arg1", "arg2", "arg3"}, []interface{}{"abcd", "efgh", "ijkl", nil})
	checkParseResults(t, parser, "optional abcd efgh ijkl mnop", nil, []string{"arg0", "arg1", "arg2", "arg3"}, []interface{}{"abcd", "efgh", "ijkl", "mnop"})
}

func TestParseBool(t *testing.T) {
	// Construct the command parser
	parser := makeTestParser()

	// Test various false values
	checkParseResults(t, parser, "test_bool abcd false 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "false", "123.345"})
	checkParseResults(t, parser, "test_bool abcd faLSe 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "false", "123.345"})
	checkParseResults(t, parser, "test_bool abcd False 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "false", "123.345"})
	checkParseResults(t, parser, "test_bool abcd FALSE 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "false", "123.345"})
	checkParseResults(t, parser, "test_bool abcd 0 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "false", "123.345"})

	// Test various true values
	checkParseResults(t, parser, "test_bool abcd true 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "true", "123.345"})
	checkParseResults(t, parser, "test_bool abcd trUE 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "true", "123.345"})
	checkParseResults(t, parser, "test_bool abcd True 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "true", "123.345"})
	checkParseResults(t, parser, "test_bool abcd TRUE 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "true", "123.345"})
	checkParseResults(t, parser, "test_bool abcd 1 123.345", nil, []string{"string", "bool", "amount"}, []interface{}{"abcd", "true", "123.345"})

	// Test invalid value
	checkParseResults(t, parser, "test_bool abcd ghjkg 123.345", util.ErrInvalidParam, []string{"string", "bool", "amount"}, []interface{}{"abcd", nil, "123.345"})

}

func checkParseResults(t *testing.T, parser *CommandParser, cmd string, errType error, names []string, values []interface{}) {
	res, err := parser.Parse(cmd)
	if errType != nil {
		assert.ErrorIs(t, err, errType)
		return
	} else if err != nil {
		assert.NoError(t, err)
	}

	for i, name := range names {
		// Get creative to parse the results
		var s *string = nil
		if values[i] != nil {
			v := values[i].(string)
			s = &v
		}

		if s == nil {
			assert.Nil(t, res.CommandResults[0].Args[name])
			return
		}

		assert.Equal(t, *s, *res.CommandResults[0].Args[name])
	}
}

// Test that parser correctly parses terminators
func TestParserTermination(t *testing.T) {
	parser := makeTestParser()

	checkTerminators(t, parser, "test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg", []TerminationStatus{InputTermination})
	checkTerminators(t, parser, "test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg;", []TerminationStatus{CommandTermination})
	checkTerminators(t, parser, "  test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg   ", []TerminationStatus{InputTermination})
	checkTerminators(t, parser, "      test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg  ;   ", []TerminationStatus{CommandTermination})
	checkTerminators(t, parser, "test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg", []TerminationStatus{NoTermination})
	checkTerminators(t, parser, "test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg; test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg", []TerminationStatus{CommandTermination, InputTermination})
	checkTerminators(t, parser, "test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg; test_address 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg;", []TerminationStatus{CommandTermination, CommandTermination})
}

func checkTerminators(t *testing.T, parser *CommandParser, input string, terminators []TerminationStatus) {
	results, err := parser.Parse(input)
	assert.NoError(t, err)
	assert.Len(t, terminators, results.Len())

	for i, result := range results.CommandResults {
		assert.Equal(t, terminators[i], result.Termination)
	}
}

func TestWalletFile(t *testing.T) {
	testKey := []byte{0x03, 0x02, 0x01, 0x0A, 0x0B, 0x0C}

	// Storage of test bytes
	file, err := ioutil.TempFile("", "wallet_test_*")
	defer os.Remove(file.Name())
	assert.NoError(t, err)

	err = util.CreateWalletFile(file, "my_password", testKey)
	assert.NoError(t, err)

	file.Close()

	// A successful retrieval of stored bytes
	file, err = os.OpenFile(file.Name(), os.O_RDONLY, 0600)
	assert.NoError(t, err)

	result, err := util.ReadWalletFile(file, "my_password")
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(result, testKey), "retrieved private key from wallet file mismatch")

	file.Close()

	// An usuccessful retrieval of stored bytes using wrong password
	file, err = os.OpenFile(file.Name(), os.O_RDONLY, 0600)
	assert.NoError(t, err)

	_, err = util.ReadWalletFile(file, "not_my_password")
	assert.Error(t, err)

	file.Close()

	// Prevent an empty passphrase
	errfile, err := ioutil.TempFile("", "wallet_test_*")
	defer os.Remove(errfile.Name())

	assert.NoError(t, err)

	err = util.CreateWalletFile(errfile, "", testKey)
	assert.ErrorIs(t, err, util.ErrEmptyPassphrase, "An empty passphrase should be disallowed")

	errfile.Close()
}

func TestParseMetrics(t *testing.T) {
	// Construct the command parser
	parser := makeTestParser()

	// Test parsing a half finished command
	checkMetrics("test_mu", parser, t, true, 0, -1, CmdNameArg)

	// Test parsing a finished command
	checkMetrics("test_multi", parser, t, true, 0, -1, CmdNameArg)

	// Test parsing a finished command with a space
	checkMetrics("test_multi ", parser, t, true, 0, 0, AddressArg)

	// Test parsing the rest of the arguments address string amount string
	checkMetrics("test_multi 16KsSj5mU", parser, t, true, 0, 0, AddressArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg", parser, t, true, 0, 0, AddressArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg ", parser, t, true, 0, 1, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword str", parser, t, true, 0, 1, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string'", parser, t, true, 0, 1, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' ", parser, t, true, 0, 2, AmountArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403", parser, t, true, 0, 2, AmountArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873", parser, t, true, 0, 2, AmountArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 ", parser, t, true, 0, 3, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str", parser, t, false, 0, 3, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str ", parser, t, false, 0, 3, StringArg) // What should happen here?
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str;", parser, t, false, 1, -1, CmdNameArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; ", parser, t, false, 1, -1, CmdNameArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; tes", parser, t, true, 1, -1, CmdNameArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; test_string", parser, t, true, 1, -1, CmdNameArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; test_string ", parser, t, true, 1, 0, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; test_string \"abc", parser, t, true, 1, 0, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; test_string \"ab' \\\"cdef\"", parser, t, false, 1, 0, StringArg)
	checkMetrics("test_multi 1GbiqgoMhvkztWytizNPn8g5SvXrrYHQQg 'a multiword string' 50.403873 basic_str; test_string \"ab' \\\"cdef\";", parser, t, false, 2, -1, CmdNameArg)

	// Test parsing invalid command followed by spaces
	checkMetrics("n  ", parser, t, true, 0, 0, NoArg)
	checkMetrics("    n         ", parser, t, true, 0, 0, NoArg)
	checkMetrics("nonsense ", parser, t, true, 0, 0, NoArg)
	checkMetrics(" a  d dsf ", parser, t, true, 0, 0, NoArg)
}

func checkMetrics(input string, parser *CommandParser, t *testing.T, expectError bool, index int, arg int, pType CommandArgType) {
	res, err := parser.Parse(input)
	if expectError {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}

	metrics := res.Metrics()

	assert.Equal(t, index, metrics.CurrentResultIndex)
	assert.Equal(t, arg, metrics.CurrentArg)
	assert.Equal(t, pType, metrics.CurrentParamType)
}
