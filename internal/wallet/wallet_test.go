package wallet

import (
	"bytes"
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

func TestParser(t *testing.T) {
	// Construct the command parser
	commands := BuildCommands()
	parser := NewCommandParser(commands)

	// Test parsing several commands
	results, err := parser.Parse("balance 1iwBq2QAax2URVqU2h878hTs8DFFKADMk; exit; quit")
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

	if results[0].CurrentArg != -1 {
		t.Error("Expected current arg to be -1, got", results[0].CurrentArg)
	}

	results, err = parser.Parse("asdasd ")
	if err == nil {
		t.Error("Expected error, got none")
	}

	if results[0].CurrentArg != 0 {
		t.Error("Expected current arg to be 0, got", results[0].CurrentArg)
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
		t.Error(fmt.Sprintf("Expected %s, got %s", address0, results[0].Args["address"]))
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
