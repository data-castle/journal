# Contributing to Journal

Thank you for your interest in contributing to Journal! This document provides guidelines and information for developers.

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- age CLI tool (optional, for testing encryption)

### Getting Started

1. Fork and clone the repository:
```bash
git clone https://github.com/yourusername/journal.git
cd journal
```

2. Install dependencies:
```bash
go mod download
```

3. Run tests:
```bash
go test ./...
```

4. Run linter:
```bash
golangci-lint run
```

## Architecture Overview

### Layer Responsibilities

1. **CLI Layer** (`cmd/journal/main.go`)
   - Parses command-line arguments
   - Handles user input (including sensitive data like passphrases)
   - Calls journal operations
   - Formats output for display

2. **Business Logic Layer** (`internal/entry/journal.go`)
   - Implements journal operations (add, search, delete, etc.)
   - Coordinates between storage, crypto, and index
   - Manages transaction-like operations (e.g., save entry + update index)

3. **Storage Layer** (`internal/storage/storage.go`)
   - Handles file system operations
   - Encrypts/decrypts entries using crypto layer
   - Manages directory structure (year/month organization)
   - Loads and saves index

4. **Crypto Layer** (SOPS integration)
   - Uses SOPS CLI for encryption/decryption
   - Each journal has `.sops.yaml` with age recipients
   - SOPS handles key discovery automatically via SOPS_AGE_KEY_FILE
   - Provides wrapper functions for encrypting/decrypting YAML files

5. **Config Layer** (`internal/config/config.go`)
   - Loads and saves journal configuration
   - Manages multiple journal configurations
   - Handles default journal selection

6. **Models Layer** (`pkg/models/`)
   - Defines data structures (Entry, Index, Metadata)
   - Implements versioned entry schema
   - Provides serialization/deserialization (YAML, JSON)

### Data Flow Example: Adding an Entry

```
User input → CLI → Journal.Add() → Storage.SaveEntry() → Crypto.Encrypt() → File system
                       ↓
                   Index.Add() → Storage.SaveIndex() → Crypto.Encrypt() → File system
```

## Key Design Decisions

### 1. Versioned Entry Schema

Entries use a versioned schema to support future changes:

```go
type Entry interface {
    GetVersion() int
    GetID() string
    GetDate() time.Time
    // ... other methods
}

type EntryV1 struct {
    MetadataV1
    Content string
}
```

This allows:
- Adding new entry types without breaking existing data
- Parsing old entries with `ParseYaml()` version detection
- Future migrations or format changes

### 2. Version-Agnostic Index

The index stores version-agnostic metadata:

```go
type Metadata struct {
    Id       string
    Date     time.Time
    Tags     []string
    FilePath string
}
```

This allows:
- Fast searches without loading full entries
- Index works with any entry version
- Entries can be upgraded without rebuilding index

### 3. SOPS with Age Encryption

The crypto layer uses SOPS for encryption:
- Users generate age keys with `age-keygen`
- Each journal has its own `.sops.yaml` with age recipients
- SOPS discovers private keys automatically via SOPS_AGE_KEY_FILE environment variable
- Supports both single-user and group journals (multiple recipients in `.sops.yaml`)

### 4. One File Per Entry

Each entry is a separate encrypted file:
- Git-friendly: fine-grained change tracking
- Parallel access: multiple entries can be read simultaneously
- Corruption isolation: one corrupted file doesn't affect others
- Efficient: only decrypt entries you need

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/models/
```

## Code Style

### General Guidelines

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write clear, self-documenting code
- Add comments for non-obvious logic

### Error Handling

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to save entry: %w", err)
}
```

### Interface Usage

Use interfaces for:
- Versioned types (e.g., `Entry` interface)
- Dependency injection in tests
- Abstracting external dependencies

Avoid interfaces for:
- Single concrete implementations with no foreseeable alternatives

## Adding New Features

### Adding a New Entry Version

1. Define new entry struct implementing `Entry` interface:
```go
type EntryV2 struct {
    MetadataV2
    Content     string
    Attachments []string // new feature
}
```

2. Implement all `Entry` interface methods

3. Update `ParseYaml()` to handle new version:
```go
case 2:
    var entry EntryV2
    if err := yaml.Unmarshal(content, &entry); err != nil {
        return nil, err
    }
    return &entry, nil
```

4. Update storage layer if needed

5. Write tests for new version

6. Consider migration path from V1 to V2

### Adding a New CLI Command

1. Add command handler function:
```go
func runMyCommand(args []string) {
    // Parse flags
    // Load journal
    // Execute operation
    // Display results
}
```

2. Register command in `main()`:
```go
case "mycommand":
    runMyCommand(os.Args[2:])
```

3. Update help text

4. Test manually and write integration tests if needed

## Debugging

### Viewing Encrypted Data

To debug encrypted entries:

```bash
# Decrypt entry manually with SOPS
cd ~/my-journal
sops -d entries/2024/11/<uuid>.yaml

# Decrypt index
sops -d index.yaml
```

### Common Issues

TBD

## Commit Guidelines

- Write clear, descriptive commit messages
- Use conventional commit format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `test:` for test additions/changes
  - `refactor:` for code refactoring
  - `chore:` for maintenance tasks

Example:
```
feat: add full-text search capability

Implements full-text search across all entries by indexing
entry content during add/update operations.
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Write/update tests
5. Run tests and linter
6. Commit your changes
7. Push to your fork
8. Open a pull request

### PR Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] Code is formatted (`gofmt`)
- [ ] New features have tests
- [ ] Documentation updated if needed
- [ ] Commit messages are clear

## Release Process

TBD

## Getting Help

TBD

## License

By contributing, you agree that your contributions will be licensed under the MIT License.