# Koinos CLI

A command-line wallet and smart-contract client for Koinos. Use it interactively,
run commands from a shell, or execute a command file.

## Build and test

Use Go 1.26 or newer. Automated checks run on the supported Go 1.26 and 1.27
release lines, on Linux and macOS. Use the latest security patch of either line.

```sh
go build -o koinos-cli ./cmd/cli
go test ./...
go test -race ./...
go vet ./...
```

The Koinos utility library still depends on the legacy Bitcoin cryptography
package. Koinos protobuf encoding also requires the pinned protobuf fork's custom
field encoders. These compatibility constraints prevent blindly upgrading those
modules or removing the replacement in `go.mod`. Test serialized transactions,
signatures, contract arguments, and wallet fixtures when changing them.

## Interactive use

```sh
./koinos-cli
```

Type `list` to see available commands and `help <command>` for arguments. Tab
completion includes registered contract commands and file paths. The prompt shows
the last verified network, an abbreviated wallet address when unlocked, and the
number of pending operations. Type `status` for the full wallet address, endpoint,
and chain identity. A successful connection check is not a continuous health check.
The prompt always uses plain text; `--force-text-prompt` remains accepted for compatibility.

```text
connect mainnet
status
disconnect
exit
```

`quit` also exits. `--network mainnet`, `--network testnet`, or `--rpc <url>` selects
an initial endpoint and verifies its chain ID, with a bounded connection wait.
Named networks reject an unexpected chain identity. `connect` also accepts these
network names or a custom URL. Once connected to a verified mainnet or testnet,
KOIN and VHP balance, supply, and transfer commands are available automatically,
including with `--no-rc` or when running outside this checkout. A custom `--rpc`
URL serving either known chain gets the same commands. Unknown chains require
explicit token registration.

Optional startup files add other tokens, contract functions, and personal settings.
Interactive startup shows each file's full path and contents, then asks
`Load this startup file? [y/N]:`. Only `y` or `yes` loads it; Enter skips it. Ctrl-C
cancels startup. Each file requires its own approval on every launch, and the CLI
executes the exact contents shown, even if the file changes during review. Control
and non-ASCII characters are escaped in the preview so they cannot hide commands.
Approved files run before `--execute` commands and the interactive prompt:

- With `--network testnet` (or `--rpc` pointing to the verified testnet), the CLI
  offers to load `.koinosrc.testnet` in your home directory and current directory. The
  included file registers testnet KOIN and VHP using their testnet addresses.
- Otherwise, the CLI offers to load `.koinosrc` in those locations. The included
  file connects to mainnet and registers KOIN, VHP, and system contracts.
- An explicit `--network` or `--rpc` takes precedence over `connect` and
  `disconnect` commands in automatic startup files; those commands are skipped
  with a notice. Other approved startup commands still run. `--no-rc` skips startup
  file discovery and review entirely. Skipping a config keeps built-in token commands.

Batch runs cannot approve discovered files. Interactive review requires terminal
input and terminal output on standard error. If review is unavailable and an
automatic startup file is present, the CLI stops before running its commands or any
`--execute`/`--file` inputs. Use `--no-rc` to skip configs, or explicitly choose a
file as shown below. Preview text goes to the terminal on standard error.

Custom startup files are loaded from disk. To use your customizations outside this
checkout, copy the appropriate file to your home directory or explicitly load
your own config and enter the prompt:

```sh
./koinos-cli --no-rc --file /path/to/my-config -i
```

Explicit `--file` inputs are deliberate script execution and do not prompt for
review. They run all their commands, including `connect`. Use contract
addresses for the network you selected; mainnet addresses do not work on testnet.

Changing connections during a transaction session is rejected. Switching to a
different chain with registered contracts (including built-in tokens) requires
restarting the CLI for that network, so old contract addresses cannot silently
carry over. Reconnecting to another node on the same chain keeps the registrations.

For testnet, use the Koinos Foundation endpoint. This read-only example checks
the chain ID without opening a wallet or submitting a transaction:

```sh
./koinos-cli --no-rc --network testnet --execute 'status'
```

The testnet profile uses `https://testnet.koinosfoundation.org/jsonrpc`.

## Wallets

```text
create my.wallet
open my.wallet
address
public
close
```

`create`, `open`, and `import` accept an optional final password argument. When it
is omitted, they use `WALLET_PASS` from the environment or a local `.env` file if
set. Otherwise, a terminal prompts for the password without displaying it.
Creating or importing a wallet asks for confirmation; opening asks once. Ctrl-C
cancels password entry without creating a wallet file. Redirected input never
prompts; scripts must supply a password argument or `WALLET_PASS`.

Wallet create/open/import commands are omitted from the CLI's interactive history.
Avoid putting real passwords or private keys in your shell history. `lock` and `unlock`
are aliases for `close` and `open`.

To import an existing Wallet Import Format (WIF) key:

```text
import <private-key-in-WIF> imported.wallet
```

New files are created with owner-only permissions and never replace an existing
file or symlink. Invalid passwords are rejected before creation, and failed writes
are cleaned up. Wallets must be open before signing or preparing write operations.

### Wallet format compatibility

New wallets use a random salt and Argon2id to derive their encryption key, then
DARE v2 authenticated encryption. The binary layout is `KOINOSW`, version byte
`0x01`, a 32-byte salt, then the encrypted private key. Version 1 fixes Argon2id to
three passes, 64 MiB of memory, four lanes, and a 32-byte derived key. File contents
cannot request larger password-derivation costs.

Existing wallets using the older SHA-256 password derivation still open. Opening
a wallet does not rewrite it or upgrade its password protection. New-format
wallets require this updated CLI and cannot be opened by older releases. Keep
existing backups when explicitly re-importing a key into a new wallet file.

`private` displays the open wallet's private key. `generate` displays a new key
without saving an encrypted wallet.

## Tokens

KOIN and VHP commands are built in for verified mainnet and testnet connections;
no config file is needed. They appear in `list`, `help`, and tab completion.

```text
koin.balance_of
koin.balance_of <address>
koin.total_supply
koin.transfer <recipient-address> 1.25
```

These token amounts are decimal values. Transfers require an open wallet and
check its current balance when online.

The `register_token` command adds the same balance, supply, and transfer commands
for another token, either interactively or in a config file. Symbol and precision
can be supplied explicitly; otherwise they are queried from the node:

```text
register_token mytoken <contract-address>
mytoken.balance_of <address>
```

Older configs that also register KOIN and VHP still work. Repeating a registration
with the same name and address reuses the existing commands; any explicit symbol
or precision must match. Conflicting registrations fail without replacing the
original commands. To load additional ABI functions for an already registered
token, use a separate contract name as shown below.

## Smart contracts

Upload compiled WebAssembly using the open wallet's address as the contract ID:

```text
upload contract.wasm contract.abi
```

Register a contract's application binary interface (ABI) from a file, or omit the
file to fetch it from the node:

```text
register mycontract <contract-address> contract.abi
help mycontract.some_method
```

The ABI generates commands with typed arguments. Read-only methods query the node;
write methods create signed transaction operations. ABI integer amounts are raw
integer units, unlike the decimal amounts accepted by `register_token` transfers.
The lower-level `call`, `read`, `set_system_call`, and `set_system_contract`
commands are also available; consult their command help.

## Transaction settings and sessions

```text
payer me
nonce auto
chain_id auto
rclimit 10%
account_rc
account_nonce
```

`payer` can select another account. `nonce` and `chain_id` can be set explicitly,
or left automatic. `rclimit` accepts an absolute mana amount or a percentage;
explicit zero is preserved and is not expanded to the available allowance.

A session batches write operations into a single transaction:

```text
session begin
koin.transfer <recipient-address> 1.25
session view
session submit
```

`session cancel` discards pending operations. Read-only commands execute immediately
and are not added to the session. A submission error retains the pending session.
Successful submission is not a guarantee of irreversible inclusion in the chain.

Offline sessions require an open wallet, an explicit nonce, an explicit base64
chain ID, and an absolute resource limit. Token registration requires symbol and
precision while offline; contract registration requires a local ABI file.
`session submit` then prints the signed transaction as JSON and base64 without
broadcasting it. `sign_transaction` adds a signature to a base64 transaction;
`submit_transaction` broadcasts one when connected.

## Scripts and exit status

For commands that do not require automatic startup configuration:

```sh
./koinos-cli --no-rc --execute 'list'
./koinos-cli --no-rc --file commands.txt
```

`--execute` accepts semicolon-separated commands and can be repeated. Command
files contain one command line per line; blank lines are accepted. To register
and immediately use a new contract in a script, put registration and use on
separate lines (or separate `--execute` options), because each line is parsed
before its commands execute.

Interactive execution order is the approved home startup file, the approved current-directory startup file,
explicit `--execute` commands, then explicit `--file` inputs. The testnet startup
filename is `.koinosrc.testnet` when testnet is selected with `--network` or verified
through `--rpc`; otherwise it is `.koinosrc`. The same automatic startup file is
reviewed and run only once. Missing automatic files are allowed; missing explicit
files are errors. Batch runs require `--no-rc` when a startup file is present;
this skips only automatic files, not explicitly named `--file` inputs.

A parse error, command error, rejected transaction, or unreadable script stops
execution and returns a nonzero process status. Later commands do not execute;
earlier completed operations are not rolled back. Successful batches return zero.
Interactive command errors return to the prompt. `--force-interactive` enters the
prompt after successful startup and batch execution.
