# Changelog

All notable changes to this project will be documented in this file.

## [v0.2.0]

### Added

- Optional parameters
- .env file handling
- error handling when attempting to transfer too low an amount
- session
- abi generated commands

### Changed

- Transfer checks balance to ensure there is enough tKOIN
- balance command will now use the open wallet's address is none is given
- open, import, and create now check the environment variable `WALLET_PASS` if no password is given
