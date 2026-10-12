package cli

import (
	"reflect"
	"testing"
)

func TestRepeatedTokenRegistrationPreservesCommands(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	command := "register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN 8"
	if result := ParseAndInterpret(ee.Parser, ee, command); result.Err != nil {
		t.Fatal(result.Err)
	}
	before := ee.Parser.Commands.List(false)
	for _, repeated := range []string{command, "register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL", "register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN 08"} {
		if result := ParseAndInterpret(ee.Parser, ee, repeated); result.Err != nil {
			t.Fatalf("matching registration failed: %v", result.Err)
		}
		if !reflect.DeepEqual(before, ee.Parser.Commands.List(false)) {
			t.Fatal("matching registration duplicated help/completion entries")
		}
	}
}

func TestConflictingTokenRegistrationPreservesOriginal(t *testing.T) {
	for _, conflict := range []string{
		"register_token coin 18tWNU7E4yuQzz7hMVpceb9ixmaWLVyQsr COIN 8",
		"register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL WRONG 8",
		"register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN 2",
		"register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN invalid",
	} {
		t.Run(conflict, func(t *testing.T) {
			ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
			if result := ParseAndInterpret(ee.Parser, ee, "register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN 8"); result.Err != nil {
				t.Fatal(result.Err)
			}
			original := ee.Parser.Commands.Name2Command["coin.balance_of"]
			if result := ParseAndInterpret(ee.Parser, ee, conflict); result.Err == nil {
				t.Fatal("conflicting registration succeeded")
			}
			if ee.Parser.Commands.Name2Command["coin.balance_of"] != original {
				t.Fatal("conflicting registration changed the original command")
			}
		})
	}
}

func TestTokenRegistrationDoesNotReplaceABIContract(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	if err := ee.Contracts.Add("coin", "15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL", &ABI{}, nil); err != nil {
		t.Fatal(err)
	}
	if result := ParseAndInterpret(ee.Parser, ee, "register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN 8"); result.Err == nil {
		t.Fatal("token registration replaced an ABI registration")
	}
}
