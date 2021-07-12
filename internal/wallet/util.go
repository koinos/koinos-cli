package wallet

import (
	"encoding/base64"

	types "github.com/koinos/koinos-types-golang"
	"github.com/shopspring/decimal"
)

func ContractStringToID(s string) (*types.ContractIDType, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	cid := types.NewContractIDType()
	if err != nil {
		return cid, err
	}

	copy(cid[:], b)
	return cid, nil
}

func KoinToDecimal(balance *types.UInt64) *decimal.Decimal {
	v := decimal.NewFromInt(int64(*balance)).Div(decimal.NewFromInt(100000000))
	return &v
}
