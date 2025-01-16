module github.com/koinos/koinos-cli

go 1.15

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/ethereum/go-ethereum v1.10.9 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/koinos/go-prompt v0.0.0-20221201222302-dba4c3542a91
	github.com/koinos/koinos-proto-golang/v2 v2.3.0
	github.com/koinos/koinos-util-golang/v2 v2.0.1
	github.com/minio/sio v0.3.0
	github.com/multiformats/go-multihash v0.1.0
	github.com/shopspring/decimal v1.3.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/ybbus/jsonrpc/v3 v3.1.1
	google.golang.org/protobuf v1.30.0
)

replace google.golang.org/protobuf => github.com/koinos/protobuf-go v1.27.2-0.20211026185306-2456c83214fe
