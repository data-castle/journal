# Journal

A secure, encrypted command-line journal application. Uses [SOPS](https://github.com/getsops/sops) with [age encryption](https://age-encryption.org/) to keep your entries private.

## Features

- Encrypted by default using SOPS + age
- Multiple journals support
- Search by date, tags, or date range
- Git-friendly
- Group journals with multiple recipients

## Prerequisites

- Go 1.25+
- [SOPS](https://github.com/getsops/sops)
- [age](https://age-encryption.org/)

## Installation

```bash
git clone https://github.com/data-castle/journal.git
cd journal
go build -o journal ./cmd/journal
```

## Quick Start

**1. Generate age key**

```bash
age-keygen -o ~/.config/sops/age/keys.txt
age-keygen -y ~/.config/sops/age/keys.txt  # Get your public key
```

**2. Set environment variable (if needed)**

```bash
# Linux/Mac
export SOPS_AGE_KEY_FILE="$HOME/.config/sops/age/keys.txt"

# Windows
$env:SOPS_AGE_KEY_FILE = "$env:USERPROFILE\.config\sops\age\keys.txt"
```

**3. Initialize journal**

```bash
journal init --name personal --path ~/my-journal --recipients age1your-public-key...
```

**4. Add entry**

```bash
journal add "Today was a great day!"
journal add "Meeting notes" -t work,meeting
```

## Usage

### Basic Commands

```bash
journal add "Entry text"              # Add entry
journal list                          # List recent entries
journal show <id>                     # Show specific entry
journal search --tag work             # Search by tag
journal search --on 2024-11-19        # Search by date
journal delete <id>                   # Delete entry
```

### Multiple Journals

```bash
journal init --name work --path ~/work-journal --recipients age1...
journal list-journals                 # List all journals
journal set-default work              # Set default journal
journal add "Text" --journal work     # Use specific journal
```

### Managing Access

```bash
journal add-recipient --name work age1newperson...     # Add recipient
journal remove-recipient --name work age1person...     # Remove recipient
journal list-recipients --name work                    # List recipients
journal re-encrypt --name work                         # Re-encrypt after changes
```

## Storage Structure

```
~/my-journal/
├── .sops.yaml              # SOPS config (recipients)
├── index.yaml              # Encrypted index
└── entries/
    └── 2024/11/
        └── <uuid>.yaml     # Encrypted entries
```

## Group Journals

Share journals by adding multiple recipients in `.sops.yaml`:

```yaml
creation_rules:
  - path_regex: .*
    age: age1alice...,age1bob...,age1charlie...
```

Anyone with their private key can decrypt. The CLI manages this automatically.

## Git Integration

```bash
cd ~/my-journal
git init
git add .
git commit -m "Initial journal"
git remote add origin <your-private-repo>
git push
```

**Note:** Keep your repo private. Entries are encrypted but metadata is visible.

## Configuration

Stored at `~/.journal/config.yaml`:

```yaml
default_journal: personal
journals:
  personal:
    name: personal
    path: /home/user/my-journal
  work:
    name: work
    path: /home/user/work-journal
```

Each journal's `.sops.yaml` manages encryption recipients.

## Security

- SOPS encrypts YAML with age (X25519 keys)
- Private keys auto-discovered via `SOPS_AGE_KEY_FILE`
- Supports single-user and group journals
- `.sops.yaml` can be committed (only has public keys)

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md)

## Status

**Implemented:**
- Core models with versioned schema
- Index system
- SOPS integration
- Configuration management
- Storage layer
- CI/CD

**TODO:**
- CLI implementation
- Entry update/edit
- Export/import
- Full-text search

## License

MIT

## Acknowledgments

- Inspired by [jrnl](https://jrnl.sh)
- [SOPS](https://github.com/getsops/sops) + [age](https://age-encryption.org/)