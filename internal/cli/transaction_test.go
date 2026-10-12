package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koinos/koinos-cli/internal/cliutil"
	kjson "github.com/koinos/koinos-proto-golang/v2/encoding/json"
	"github.com/koinos/koinos-proto-golang/v2/koinos/protocol"
	chainrpc "github.com/koinos/koinos-proto-golang/v2/koinos/rpc/chain"
	util "github.com/koinos/koinos-util-golang/v2"
	"google.golang.org/protobuf/proto"
)

func captureSubmission(t *testing.T, limit uint64) *protocol.Transaction {
	t.Helper()
	var captured *protocol.Transaction
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		var result proto.Message
		switch request.Method {
		case cliutil.GetChainIDCall:
			result = &chainrpc.GetChainIdResponse{ChainId: []byte("server-chain")}
		case cliutil.GetAccountRcCall:
			result = &chainrpc.GetAccountRcResponse{Rc: 100000000}
		case cliutil.SubmitTransactionCall:
			p := &chainrpc.SubmitTransactionRequest{}
			if err := kjson.Unmarshal(request.Params, p); err != nil {
				t.Error(err)
				return
			}
			captured = p.Transaction
			result = &chainrpc.SubmitTransactionResponse{Receipt: &protocol.TransactionReceipt{Id: captured.Id}}
		default:
			t.Errorf("unexpected RPC %s", request.Method)
			return
		}
		data, err := kjson.Marshal(result)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, request.ID, data)
	}))
	defer server.Close()
	ee := NewExecutionEnvironment(cliutil.NewKoinosRPCClient(server.URL), nil)
	keyBytes := make([]byte, 32)
	keyBytes[31] = 1
	key, err := util.NewKoinosKeyFromBytes(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	ee.OpenWallet(key)
	ee.nonceMode = "1"
	ee.rcLimit = rcInfo{absolute: true, value: limit}
	ee.chainID = base64.URLEncoding.EncodeToString([]byte("chosen-chain"))
	op := &protocol.Operation{Op: &protocol.Operation_SetSystemContract{SetSystemContract: &protocol.SetSystemContractOperation{ContractId: key.AddressBytes()}}}
	if err := ee.SubmitTransaction(context.Background(), NewExecutionResult(), op); err != nil {
		t.Fatal(err)
	}
	if captured == nil {
		t.Fatal("no captured transaction")
	}
	return captured
}

func TestSubmissionHonorsManualChainID(t *testing.T) {
	tx := captureSubmission(t, 100)
	if !bytes.Equal(tx.Header.ChainId, []byte("chosen-chain")) {
		t.Fatalf("configured chosen-chain but submitted %q", tx.Header.ChainId)
	}
}
func TestSubmissionHonorsZeroRCLimit(t *testing.T) {
	tx := captureSubmission(t, 0)
	if tx.Header.RcLimit != 0 {
		t.Fatalf("configured zero mana limit but submitted limit %d", tx.Header.RcLimit)
	}
}
