package cliutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kjson "github.com/koinos/koinos-proto-golang/v2/encoding/json"
	"github.com/koinos/koinos-proto-golang/v2/koinos/protocol"
	chainrpc "github.com/koinos/koinos-proto-golang/v2/koinos/rpc/chain"
	util "github.com/koinos/koinos-util-golang/v2"
	"google.golang.org/protobuf/proto"
)

func rpcFixture(t *testing.T, respond func(string, json.RawMessage) proto.Message) *KoinosRPCClient {
	t.Helper()
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
		result := respond(request.Method, request.Params)
		data, err := kjson.Marshal(result)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, request.ID, data)
	}))
	t.Cleanup(server.Close)
	return NewKoinosRPCClient(server.URL)
}

func TestSubmissionDefaultsOnlyWhenParametersOmitted(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) {
			var submitted *protocol.Transaction
			lookups := 0
			client := rpcFixture(t, func(method string, params json.RawMessage) proto.Message {
				switch method {
				case GetAccountNonceCall:
					lookups++
					nonce, _ := util.UInt64ToNonceBytes(7)
					return &chainrpc.GetAccountNonceResponse{Nonce: nonce}
				case GetAccountRcCall:
					lookups++
					return &chainrpc.GetAccountRcResponse{Rc: 123}
				case GetChainIDCall:
					lookups++
					return &chainrpc.GetChainIdResponse{ChainId: []byte("server")}
				case SubmitTransactionCall:
					request := &chainrpc.SubmitTransactionRequest{}
					if err := kjson.Unmarshal(params, request); err != nil {
						t.Error(err)
					}
					submitted = request.Transaction
					return &chainrpc.SubmitTransactionResponse{Receipt: &protocol.TransactionReceipt{}}
				default:
					t.Errorf("unexpected method %s", method)
					return &chainrpc.GetChainIdResponse{}
				}
			})
			keyBytes := make([]byte, 32)
			keyBytes[31] = 1
			key, err := util.NewKoinosKeyFromBytes(keyBytes)
			if err != nil {
				t.Fatal(err)
			}
			var params *SubmissionParams
			if explicit {
				params = &SubmissionParams{Nonce: 0, RCLimit: 0, ChainID: []byte("chosen")}
			}
			op := &protocol.Operation{Op: &protocol.Operation_SetSystemContract{SetSystemContract: &protocol.SetSystemContractOperation{ContractId: key.AddressBytes()}}}
			if _, err := client.SubmitTransactionOps(context.Background(), []*protocol.Operation{op}, key, params, true); err != nil {
				t.Fatal(err)
			}
			if submitted == nil {
				t.Fatal("no submission")
			}
			nonce, err := util.NonceBytesToUInt64(submitted.Header.Nonce)
			if err != nil {
				t.Fatal(err)
			}
			if explicit {
				if lookups != 0 || nonce != 0 || submitted.Header.RcLimit != 0 || string(submitted.Header.ChainId) != "chosen" {
					t.Fatal("explicit zero settings were replaced")
				}
			} else if lookups != 3 || nonce != 8 || submitted.Header.RcLimit != 123 || string(submitted.Header.ChainId) != "server" {
				t.Fatal("automatic settings were not fetched")
			}
		})
	}
}

func TestSubmissionRejectsRevertedOrMissingReceipt(t *testing.T) {
	for _, receipt := range []*protocol.TransactionReceipt{nil, {Reverted: true, Logs: []string{"contract rejected operation"}}} {
		client := rpcFixture(t, func(string, json.RawMessage) proto.Message {
			return &chainrpc.SubmitTransactionResponse{Receipt: receipt}
		})
		_, err := client.SubmitTransaction(context.Background(), &protocol.Transaction{}, true)
		if err == nil {
			t.Fatalf("reported success for receipt %+v", receipt)
		}
	}
}
