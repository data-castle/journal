package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/entry"
)

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	name := fs.String("name", "", "Journal name (required)")
	fs.StringVar(name, "n", "", "Journal name (shorthand)")
	path := fs.String("path", "", "Custom path for journal (required)")
	fs.StringVar(path, "p", "", "Custom path for journal (shorthand)")
	recipients := fs.String("recipients", "", "Age public keys (comma-separated, required)")
	fs.StringVar(recipients, "r", "", "Age public keys (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal init --name <name> --path <path> --recipients <keys>")
		fmt.Println("\nInitialize a new journal with SOPS encryption")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
		fmt.Println("\nExample:")
		fmt.Println("  journal init -n work -p ~/work-journal -r age1key1...,age1key2...")
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *name == "" {
		if _, err := fmt.Fprintf(os.Stderr, "Error: --name is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}
	if *path == "" {
		if _, err := fmt.Fprintf(os.Stderr, "Error: --path is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}
	if *recipients == "" {
		if _, err := fmt.Fprintf(os.Stderr, "Error: --recipients is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}

	recipientKeys := strings.Split(*recipients, ",")
	for i := range recipientKeys {
		recipientKeys[i] = strings.TrimSpace(recipientKeys[i])
	}

	journalPath := *path
	if strings.HasPrefix(journalPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			if _, ferr := fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err); ferr != nil {
				return 1
			}
			return 1
		}
		journalPath = filepath.Join(homeDir, journalPath[1:])
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	journalCfg := &config.Journal{
		Name: *name,
		Path: journalPath,
	}

	if err := entry.InitializeJournal(journalCfg, recipientKeys); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to initialize journal: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	// Check if journal with this name already exists
	if existingJournal, exists := cfg.Journals[*name]; exists {
		if _, err := fmt.Fprintf(os.Stderr, "Warning: A journal named '%s' already exists at %s\n", *name, existingJournal.Path); err != nil {
			return 1
		}
		if _, err := fmt.Fprintf(os.Stderr, "Updating journal location to: %s\n", journalPath); err != nil {
			return 1
		}
		// Update the existing journal's path
		existingJournal.Path = journalPath
	} else {
		// Add new journal
		if err := cfg.AddJournal(journalCfg); err != nil {
			if _, ferr := fmt.Fprintf(os.Stderr, "Failed to add journal to config: %v\n", err); ferr != nil {
				return 1
			}
			return 1
		}
	}

	if err := cfg.Save(); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Journal '%s' initialized at %s\n", *name, journalPath); err != nil {
		return 1
	}
	if _, err := fmt.Printf("Recipients: %d\n", len(recipientKeys)); err != nil {
		return 1
	}
	if _, err := fmt.Println("\nNext steps:"); err != nil {
		return 1
	}
	if _, err := fmt.Println("1. Ensure SOPS_AGE_KEY_FILE environment variable is set"); err != nil {
		return 1
	}
	if _, err := fmt.Println("2. (Optional) Initialize git:"); err != nil {
		return 1
	}
	if _, err := fmt.Printf("   cd %s && git init\n", journalPath); err != nil {
		return 1
	}
	if _, err := fmt.Printf("3. Start adding entries: journal add \"Your first entry\"\n"); err != nil {
		return 1
	}
	return 0
}
