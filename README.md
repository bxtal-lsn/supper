# supper

A terminal user interface for managing encrypted secrets with SOPS and age.

## Features

- Generate and manage age encryption keys
- Encrypt, decrypt, and edit files with SOPS
- Handle key rotation and multiple recipients
- Secure workflow for managing encrypted secrets

## Installation

```bash
go install github.com/bxtal-lsn/supper/cmd/supper@latest
```

## Usage

```bash
supper
```

## Requirements

- SOPS (`sops`) command-line tool
- age (`age` and `age-keygen`) command-line tools

## Development

```bash
# Clone the repositor
git clone https://github.com/bxtal-lsn/supper.git
cd supper

# Build the application
go build -o supper ./cmd/supper

# Run the application
./supper
```
