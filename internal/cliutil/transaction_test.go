package cliutil

import (
	"context"
	"encoding/hex"
	"github.com/koinos/koinos-proto-golang/v2/koinos/protocol"
	util "github.com/koinos/koinos-util-golang/v2"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestSignedTransactionMatchesLegacyFixture(t *testing.T) {
	raw := make([]byte, 32)
	raw[31] = 1
	key, err := util.NewKoinosKeyFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	op := &protocol.Operation{Op: &protocol.Operation_SetSystemContract{SetSystemContract: &protocol.SetSystemContractOperation{ContractId: key.AddressBytes(), SystemContract: true}}}
	tx, err := CreateSignedTransaction(context.Background(), []*protocol.Operation{op}, key, 7, 100000000, []byte("fixture-chain"), key.AddressBytes())
	if err != nil {
		t.Fatal(err)
	}
	b, err := proto.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	// Captured using the original dependency set; protects wire format and signing.
	want := "0a2212200fc515c79c1203315c5a3569406e584ff6b9652ca0c43fd12fe4faccd378cd9712570a0d666978747572652d636861696e1080c2d72f1a022807222212206de94032ac3b8951ee5b23b508ceaf41cc812b7a0626eb4be756d5575358db882a1900751e76e8199196d454941c45d1b3a323f1433bd6510d16341a1f221d0a1900751e76e8199196d454941c45d1b3a323f1433bd6510d163410012241203caab22ad87344ee501ee549bc4aa562be5de0a8867ba08671c4e1cde3cc239d09f74596cce3124f8dfa880f1aff378bb309a5f29ca579be5b8a4cce5ae1f4f1"
	if hex.EncodeToString(b) != want {
		t.Fatal("signed transaction changed after dependency update")
	}
}
