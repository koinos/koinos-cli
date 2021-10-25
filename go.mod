module github.com/koinos/koinos-cli-wallet

go 1.15

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/c-bata/go-prompt v0.2.6-0.20210301180729-8aae7fb1a6f9
	github.com/ethereum/go-ethereum v1.10.9
	github.com/joho/godotenv v1.3.0
	github.com/koinos/koinos-proto-golang v0.0.0-20211021190958-179635a3c9bd
	github.com/koinos/koinos-util-golang v0.0.0-20210909225633-da2fb456bade
	github.com/minio/sio v0.3.0
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multihash v0.0.16
	github.com/shopspring/decimal v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	google.golang.org/protobuf v1.27.1
)

replace google.golang.org/protobuf => github.com/koinos/protobuf-go v1.27.2-0.20211016005428-adb3d63afc5e
