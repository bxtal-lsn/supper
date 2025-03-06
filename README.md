# supper

A terminal user interface for managing encrypted secrets with SOPS and age.

## Features

- Generate and manage age encryption keys with passphrase protection
- Encrypt, decrypt, and edit files with SOPS
- Browse files and directories with integrated file browser
- Handle key rotation and multiple recipients
- Secure workflow for managing encrypted secrets
- Auto-deletion of decrypted keys after configurable time
- Customizable settings

## Screenshots

(Screenshots will be added when available)

## Installation

### Prerequisites

- [SOPS](https://github.com/getsops/sops) - Secrets management tool
- [age](https://github.com/FiloSottile/age) - Modern encryption tool
- Go 1.18+ (for building from source)

#### Installing Prerequisites

On macOS:
```bash
brew install sops age
```

On Linux (Ubuntu/Debian):
```bash
# For SOPS
wget https://github.com/getsops/sops/releases/download/v3.8.1/sops-v3.8.1.linux.amd64 -O /usr/local/bin/sops
chmod +x /usr/local/bin/sops

# For age
sudo apt-get update
sudo apt-get install age
```

### Installing supper

#### From Source

```bash
# Clone the repository
git clone https://github.com/bxtal-lsn/supper.git
cd supper

# Build and install
make install
```

#### Using Go Install

```bash
go install github.com/bxtal-lsn/supper/cmd/supper@latest
```

## Usage

Simply run the application:

```bash
supper
```

### Basic Workflow

1. **Generate an Age Key**: Navigate to the Key Manager tab and press `g` to generate a new key
2. Enter a strong passphrase to protect your key
3. **Work with Files**: Navigate to the Files tab and browse to your files
4. Select a file and use the following actions:
   - `e` - Encrypt a file
   - `d` - Decrypt a file
   - `E` - Edit an encrypted file

### Key Management

- Generated keys are stored encrypted with your passphrase
- Decrypted keys are automatically deleted after a configurable time (default: 30 minutes)
- You can manually delete decrypted keys by pressing `x` in the Key Manager tab

## Project Structure

```
supper/
├── cmd/
│   └── supper/        # Application entry point
├── internal/
│   ├── age/             # Age key management functionality
│   ├── sops/            # SOPS integration
│   ├── config/          # Configuration management
│   ├── ui/              # User interface components
│   │   ├── views/       # Main application views
│   │   ├── components/  # Reusable UI components
│   │   └── styles/      # UI styling
│   └── utils/           # Utility functions
├── pkg/                 # Public packages
├── docs/                # Documentation
└── assets/              # Icons and other assets
```

## Development

```bash
# Clone the repository
git clone https://github.com/bxtal-lsn/supper.git
cd supper

# Install dependencies
make deps

# Build the application
make build

# Run the application
make run

# Run tests
make test

# Format code
make fmt
```

## Configuration

The application stores its configuration in:
- `~/.config/supper/config.json` (Linux/macOS)
- `%APPDATA%\supper\config.json` (Windows)

## Security Considerations

- The application securely handles decrypted keys and cleans them from memory
- All decrypted keys are securely deleted when no longer needed
- The application never stores unencrypted secrets on disk except during editing

## License

MIT

## Acknowledgements

- [SOPS](https://github.com/getsops/sops) - Mozilla's Secrets OPerationS
- [age](https://github.com/FiloSottile/age) - A simple, modern and secure file encryption tool
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal applications
