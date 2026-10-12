module github.com/koinos/koinos-cli

go 1.26.0

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/joho/godotenv v1.5.1
	github.com/koinos/go-prompt v0.0.0-20221201235110-5ab92fe3bbe2
	github.com/koinos/koinos-proto-golang/v2 v2.6.0
	github.com/koinos/koinos-util-golang/v2 v2.0.1
	github.com/minio/sio v0.5.1
	github.com/multiformats/go-multihash v0.2.3
	github.com/shopspring/decimal v1.5.0
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.12.1
	github.com/ybbus/jsonrpc/v3 v3.1.9
	golang.org/x/crypto v0.58.0
	golang.org/x/sys v0.49.0
	golang.org/x/term v0.47.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20251001021608-1fe7b43fc4d6 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ethereum/go-ethereum v1.17.8 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/pkg/term v1.1.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	lukechampine.com/blake3 v1.1.6 // indirect
)

replace google.golang.org/protobuf => github.com/koinos/protobuf-go v1.27.2-0.20211026185306-2456c83214fe
