# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0]

### What's Changed
* Change endpoint in readme to be https by @jredbeard in https://github.com/koinos/koinos-cli/pull/137
* Adds the command `get_account_rc` by @sgerbino in https://github.com/koinos/koinos-cli/pull/147
* Implementation of `register_token` by @sgerbino in https://github.com/koinos/koinos-cli/pull/148
* Rename `get_account_rc` -> `account_rc` by @mvandeberg in https://github.com/koinos/koinos-cli/pull/150
* Remove the double spacing of `.koinosrc` execution results by @sgerbino in https://github.com/koinos/koinos-cli/pull/156
* Do not display a warning when the .env file does not exist by @sgerbino in https://github.com/koinos/koinos-cli/pull/154
* Add issue management workflows by @mvandeberg in https://github.com/koinos/koinos-cli/pull/160
* Issue management by @mvandeberg in https://github.com/koinos/koinos-cli/pull/161
* #122: Handle enums and fix panic on empty list in return by @mvandeberg in https://github.com/koinos/koinos-cli/pull/162
* Involves #166 by @youkaicountry in https://github.com/koinos/koinos-cli/pull/167
* #121 Add build instructions to readme by @jredbeard in https://github.com/koinos/koinos-cli/pull/165
* Offline transaction signing by @youkaicountry in https://github.com/koinos/koinos-cli/pull/164
* Fix balance check on token transfer by @youkaicountry in https://github.com/koinos/koinos-cli/pull/171
* Allow offline signing for contract calls by @mvandeberg in https://github.com/koinos/koinos-cli/pull/172
* Add all system contracts to the default `.koinosrc` file by @sgerbino in https://github.com/koinos/koinos-cli/pull/175
* Do not check user balance upon transfer when wallet is offline by @mvandeberg in https://github.com/koinos/koinos-cli/pull/178
* Remove extra padding around `chain_id` command output by @sgerbino in https://github.com/koinos/koinos-cli/pull/179
* Crash on contract read by @youkaicountry in https://github.com/koinos/koinos-cli/pull/181
* Use uint64 to track rclimit by @youkaicountry in https://github.com/koinos/koinos-cli/pull/180
* Update github workflows by @mvandeberg in https://github.com/koinos/koinos-cli/pull/185
* Change default RC limit to 10%, show more helpful error messages by @youkaicountry in https://github.com/koinos/koinos-cli/pull/184
* Bump version number to v2.0.0 by @youkaicountry in https://github.com/koinos/koinos-cli/pull/188

**Full Changelog**: https://github.com/koinos/koinos-cli/compare/v1.0.0...v2.0.0

## [1.0.0]

### Added

- `payer` command which sets the current transaction payer

## [v0.4.0]

### Added

- `public` command which shows open wallet's public key
- `generate` now shows public key

### Changed

- Fix terminal scrolling error
- Many small parser fixes

## [v0.2.0]

### Added

- Optional parameters
- .env file handling
- error handling when attempting to transfer too low an amount
- session
- abi generated commands

### Changed

- Transfer checks balance to ensure there is enough KOIN
- balance command will now use the open wallet's address is none is given
- open, import, and create now check the environment variable `WALLET_PASS` if no password is given
