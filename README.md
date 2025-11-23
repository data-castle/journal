# Journal

A secure, encrypted command-line journal application built with Go. Uses [SOPS](https://github.com/getsops/sops) with [age encryption](https://age-encryption.org/) to keep your journal entries private.

## Features

- Encrypted by default using SOPS with age
- Support for multiple journals
- Fast search by date, date range, and tags
- Git-friendly: each entry is a separate encrypted file
- Versioned entry schema for future extensibility
- Simple command-line interface
- Structured encryption for YAML entries

## Installation

### Prerequisites

- Go 1.25 or later
- [SOPS](https://github.com/getsops/sops) CLI tool
- age CLI tool (for key generation)

### Build from source

```bash
git clone https://github.com/data-castle/journal.git
cd journal
go build -o journal ./cmd/journal
```

## Quick Start

### 1. Generate age identity and configure SOPS

```bash
# Generate age key pair
age-keygen -o ~/.config/sops/age/keys.txt

# Get your public key
age-keygen -y ~/.config/sops/age/keys.txt
# Output: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Configure SOPS to find your private key (if not using default location)
# Linux/Mac:
export SOPS_AGE_KEY_FILE="$HOME/.config/sops/age/keys.txt"
# Windows:
$env:SOPS_AGE_KEY_FILE = "$env:USERPROFILE\.config\sops\age\keys.txt"
```

### 2. Initialize a journal

```bash
# Single-user journal (only you can access)
journal init --name personal --path ~/my-journal \
  --recipients age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Group journal (multiple people can access)
journal init --name work --path ~/work-journal \
  --recipients age1alice...,age1bob...,age1charlie...
```

This will:
- Create the journal directory structure
- Create `.sops.yaml` in the journal directory with the specified recipients
- Set it as your default journal (if first journal)
- Create an encrypted index

### 3. Add your first entry

```bash
journal add "Today was a great day!"

# With tags
journal add "Had a productive meeting" -t work,meeting
```

### 4. List recent entries

```bash
journal list
```

## Usage

### Managing Journals

```bash
# Initialize a new journal
journal init --name personal --path ~/my-journal \
  --recipients age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# List all configured journals
journal list-journals

# Set default journal
journal set-default personal

# Use a specific journal (instead of default)
journal add "Entry text" --journal work
```

### Managing Recipients (Access Control)

```bash
# Add a recipient to an existing journal (allows them to decrypt entries)
journal add-recipient --name work age1newperson...

# Remove a recipient from a journal
journal remove-recipient --name work age1oldperson...

# List recipients for a journal
journal list-recipients --name work

# Re-encrypt all entries after recipient changes
journal re-encrypt --name work
```

### Adding Entries

```bash
# Simple entry
journal add "Your entry text here"

# Entry with tags
journal add "Meeting notes" -t work,meeting,project-x

# Use specific journal
journal add "Work note" --journal work
```

### Viewing Entries

```bash
# List recent entries (default: 10)
journal list

# List more entries
journal list -n 20

# Show specific entry by ID
journal show <entry-id>
```

### Searching

```bash
# Search by specific date
journal search --on 2024-11-19

# Search by date range
journal search --from 2024-11-01 --to 2024-11-30

# Search last N days
journal search --last 7

# Search by tag
journal search --tag work

# Search by multiple tags (AND operation)
journal search --tags work,meeting
```

### Maintenance

```bash
# Rebuild index from entry files
journal rebuild

# Delete an entry
journal delete <entry-id>
```

## Git Integration

Journal works seamlessly with Git for backup and sync:

```bash
cd ~/my-journal
git init
git add .
git commit -m "Initial journal"
git remote add origin <your-private-repo-url>
git push -u origin main
```

## Security

- All entries are encrypted using SOPS with age encryption (X25519 key pairs)
- SOPS provides structured encryption - only sensitive fields are encrypted
- Index is also encrypted with SOPS
- Supports single-user journals (one identity) and group journals (multiple recipients via `.sops.yaml`)
- Private keys should be kept secure and never committed to version control
- `.sops.yaml` configuration can be committed to Git (it only contains public keys)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## What's Done

- [x] Core entry models with versioned schema
- [x] Index system for fast searches
- [x] Configuration management
- [x] Storage layer with year/month organization
- [x] CI/CD (tests, linting, Dependabot)

## TODO

- [ ] Migrate from direct age to SOPS integration
- [ ] Complete CLI implementation for all features
- [ ] Entry update/edit functionality
- [ ] Automatic Git integration
- [ ] Full-text search across all entries

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Inspired by [jrnl](https://jrnl.sh)
- Encrypted with [SOPS](https://github.com/getsops/sops) and [age](https://age-encryption.org/)