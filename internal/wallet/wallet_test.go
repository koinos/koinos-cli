package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	types "github.com/koinos/koinos-types-golang"
	"github.com/shopspring/decimal"
)

func TestSatoshiToDecimal(t *testing.T) {
	v, err := SatoshiToDecimal(100000000, 8)
	if err != nil {
		t.Error(err)
	}

	if !v.Equal(decimal.NewFromFloat(1.0)) {
		t.Error("Expected 1.0, got", v)
	}

	v, err = SatoshiToDecimal(1000, 1)
	if err != nil {
		t.Error(err)
	}

	if !v.Equal(decimal.NewFromFloat(100.0)) {
		t.Error("Expected 100.0, got", v)
	}

	v, err = SatoshiToDecimal(12345678, 3)
	if err != nil {
		t.Error(err)
	}

	if !v.Equal(decimal.NewFromFloat(12345.678)) {
		t.Error("Expected 1234.5678, got", v)
	}
}

func makeTestParser() *CommandParser {
	// Construct the command parser
	var decls []*CommandDeclaration
	decls = append(decls, NewCommandDeclaration("test_address", "Test command which takes an address", false, nil, *NewCommandArg("address", Address)))
	decls = append(decls, NewCommandDeclaration("test_string", "Test command which takes a string", false, nil, *NewCommandArg("string", String)))
	decls = append(decls, NewCommandDeclaration("test_none", "Test command which takes no arguments", false, nil))
	decls = append(decls, NewCommandDeclaration("test_none2", "Another test command which takes no arguments", false, nil))
	decls = append(decls, NewCommandDeclaration("test_multi", "Test command which takes multiple arguments, and of different types", false, NewGenerateKeyCommand))

	parser := NewCommandParser(decls)

	return parser
}

func TestBasicParser(t *testing.T) {
	parser := makeTestParser()

	// Test parsing several commands
	results, err := parser.Parse("test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk; test_none; test_none2")
	if err != nil {
		t.Error(err)
	}

	if len(results) != 3 {
		t.Error("Expected 3 result, got", len(results))
	}

	results, err = parser.Parse("asdasd")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if !errors.Is(err, ErrUnknownCommand) {
		t.Error("Expected error", ErrUnknownCommand, ", got", err)
	}

	if results[0].CurrentArg != -1 {
		t.Error("Expected current arg to be -1, got", results[0].CurrentArg)
	}

	results, err = parser.Parse("asdasd ")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if !errors.Is(err, ErrUnknownCommand) {
		t.Error("Expected error", ErrUnknownCommand, ", got", err)
	}

	if results[0].CurrentArg != 0 {
		t.Error("Expected current arg to be 0, got", results[0].CurrentArg)
	}

	// Test parsing empty inputs
	results, err = parser.Parse("")
	if err != nil {
		t.Error(err)
	}

	if len(results) != 0 {
		t.Error("Expected 0 results, got", len(results))
	}

	results, err = parser.Parse("    ")
	if err != nil {
		t.Error(err)
	}

	if len(results) != 0 {
		t.Error("Expected 0 results, got", len(results))
	}

	// Test nonsensical string of empty commands
	results, err = parser.Parse(" ; ;; ;; ;;;;    ;     ;  ;    ")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if len(results) != 0 {
		t.Error("Expected 0 results, got", len(results))
	}

	// Test valid command followed by empty commands
	results, err = parser.Parse("test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk;  ;;  ; ;; test_none")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if !errors.Is(err, ErrEmptyCommandName) {
		t.Error("Expected error", ErrEmptyCommandName, ", got", err)
	}

	if len(results) != 1 {
		t.Error("Expected 1 result, got", len(results))
	}
}

// Test that parser correctly parses terminators
func TestParserTermination(t *testing.T) {
	parser := makeTestParser()

	checkTerminators(t, parser, "test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk", []TerminationStatus{Input})
	checkTerminators(t, parser, "test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk;", []TerminationStatus{Command})
	checkTerminators(t, parser, "  test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk   ", []TerminationStatus{Input})
	checkTerminators(t, parser, "      test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk  ;   ", []TerminationStatus{Command})
	checkTerminators(t, parser, "test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk", []TerminationStatus{None})
	checkTerminators(t, parser, "test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk; test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk", []TerminationStatus{Command, Input})
	checkTerminators(t, parser, "test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk; test_address 1iwBq2QAax2URVqU2h878hTs8DFFKADMk;", []TerminationStatus{Command, Command})
}

func checkTerminators(t *testing.T, parser *CommandParser, input string, terminators []TerminationStatus) {
	results, err := parser.Parse(input)
	if err != nil {
		t.Error(err)
	}

	if len(results) != len(terminators) {
		t.Error("Expected", len(terminators), "results, got", len(results))
	}

	for i, result := range results {
		if result.Termination != terminators[i] {
			t.Error("Expected terminator", terminators[i], "got", result.Termination)
		}
	}
}

func TestBalance(t *testing.T) {
	// Construct the command parser
	commands := BuildCommands()
	parser := NewCommandParser(commands)

	// Test parsing a single balance command
	address0 := "1iwBq2QAax2URVqU2h878hTs8DFFKADMk"
	results, err := parser.Parse(fmt.Sprintf("balance %s", address0))
	if err != nil {
		t.Error(err)
	}

	if len(results) != 1 {
		t.Error("Expected 1 result, got", len(results))
	}

	if results[0].CommandName != "balance" {
		t.Error("Expected balance parse result, got", results[0].CommandName)
	}

	if results[0].Args["address"] != address0 {
		t.Errorf("Expected %s, got %s", address0, results[0].Args["address"])
	}

	// Test the command object instantiation
	cmd := results[0].Instantiate()
	bcmd := cmd.(*BalanceCommand)

	// Make sure the account type object is correct
	addr := types.AccountType(address0)
	if !bytes.Equal(addr, []byte(*bcmd.Address)) {
		t.Error("Address in balance command object does not match given address")
	}
}

func TestExit(t *testing.T) {
	// Construct the command parser
	commands := BuildCommands()
	parser := NewCommandParser(commands)

	// Test parsing a single balance command
	results, err := parser.Parse("quit; exit")
	if err != nil {
		t.Error(err)
	}

	if len(results) != 2 {
		t.Error("Expected 2 result, got", len(results))
	}

	if results[0].CommandName != "quit" || results[1].CommandName != "exit" {
		t.Error("Invalid command name")
	}

	if len(results[0].Args) != 0 || len(results[1].Args) != 0 {
		t.Error("Invalid exit args")
	}
}

func TestWalletFile(t *testing.T) {
	testKey := []byte{0x03, 0x02, 0x01, 0x0A, 0x0B, 0x0C}

	// Storage of test bytes
	file, err := ioutil.TempFile("", "wallet_test_*")
	defer os.Remove(file.Name())

	if err != nil {
		t.Error(err.Error())
	}

	err = CreateWalletFile(file, "my_password", testKey)

	if err != nil {
		t.Error(err.Error())
	}

	file.Close()

	// A successful retrieval of stored bytes
	file, err = os.OpenFile(file.Name(), os.O_RDONLY, 0600)

	if err != nil {
		t.Error(err.Error())
	}

	result, err := ReadWalletFile(file, "my_password")

	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(testKey, result) {
		t.Error("retrieved private key from wallet file mismatch")
	}

	file.Close()

	// An usuccessful retrieval of stored bytes using wrong password
	file, err = os.OpenFile(file.Name(), os.O_RDONLY, 0600)

	if err != nil {
		t.Error(err.Error())
	}

	_, err = ReadWalletFile(file, "not_my_password")

	if err == nil {
		t.Error("retrieved private key without correct passphrase")
	}

	file.Close()

	// Prevent an empty passphrase
	errfile, err := ioutil.TempFile("", "wallet_test_*")
	defer os.Remove(errfile.Name())

	if err != nil {
		t.Error(err.Error())
	}

	err = CreateWalletFile(errfile, "", testKey)

	if err != ErrEmptyPassphrase {
		t.Error("an empty passphrase is not allowed")
	}

	errfile.Close()
}
