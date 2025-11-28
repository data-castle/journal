package cli

import (
	"fmt"
	"os"

	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/crypto"
)

func runListJournals(args []string) int {
	cfg, err := config.LoadConfig()
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if len(cfg.Journals) == 0 {
		if _, err := fmt.Println("No journals configured"); err != nil {
			return 1
		}
		if _, err := fmt.Println("\nUse 'journal init' to create a journal"); err != nil {
			return 1
		}
		return 0
	}

	if _, err := fmt.Println("Configured journals:"); err != nil {
		return 1
	}
	for name, j := range cfg.Journals {
		marker := ""
		if name == cfg.DefaultJournal {
			marker = " (default)"
		}
		if _, err := fmt.Printf("\n  %s%s\n", name, marker); err != nil {
			return 1
		}
		if _, err := fmt.Printf("    Path: %s\n", j.Path); err != nil {
			return 1
		}

		recipients, err := crypto.ReadSOPSConfig(j.Path)
		if err == nil {
			if _, err := fmt.Printf("    Recipients: %d\n", len(recipients)); err != nil {
				return 1
			}
		}
	}
	return 0
}

func runSetDefault(args []string) int {
	if len(args) == 0 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: journal name is required\n"); err != nil {
			return 1
		}
		if _, err := fmt.Fprintf(os.Stderr, "Usage: journal set-default <name>\n"); err != nil {
			return 1
		}
		return 1
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if err := cfg.SetDefaultJournal(args[0]); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to set default: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if err := cfg.Save(); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Default journal set to: %s\n", args[0]); err != nil {
		return 1
	}
	return 0
}
