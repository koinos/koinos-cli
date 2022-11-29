module github.com/koinos/koinos-cli

go 1.15

require (
	github.com/btcsuite/btcutil v1.0.2
	github.com/ethereum/go-ethereum v1.10.9 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/koinos/go-prompt v0.0.0-20220818181004-5b1028a45a2f
	github.com/koinos/koinos-proto-golang v1.0.1-0.20221123003957-336b725f600d
	github.com/koinos/koinos-util-golang v1.0.1-0.20221129203044-ed05bd389c58
	github.com/minio/sio v0.3.0
	github.com/shopspring/decimal v1.3.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	google.golang.org/protobuf v1.27.1
)

replace google.golang.org/protobuf => github.com/koinos/protobuf-go v1.27.2-0.20211026185306-2456c83214fe
