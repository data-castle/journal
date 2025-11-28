package cli

import (
	"fmt"
	"os"

	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/entry"
)

var Version = "1.0.0"

func Run(args []string) int {
	if len(args) < 2 {
		printUsage()
		return 1
	}

	cmd := args[1]
	cmdArgs := args[2:]

	switch cmd {
	case "init":
		return runInit(cmdArgs)
	case "add":
		return runAdd(cmdArgs)
	case "list":
		return runList(cmdArgs)
	case "search":
		return runSearch(cmdArgs)
	case "show":
		return runShow(cmdArgs)
	case "delete":
		return runDelete(cmdArgs)
	case "rebuild":
		return runRebuild(cmdArgs)
	case "list-journals":
		return runListJournals(cmdArgs)
	case "set-default":
		return runSetDefault(cmdArgs)
	case "add-recipient":
		return runAddRecipient(cmdArgs)
	case "remove-recipient":
		return runRemoveRecipient(cmdArgs)
	case "re-encrypt":
		return runReEncrypt(cmdArgs)
	case "help", "-h", "--help":
		printUsage()
		return 0
	case "version", "-v", "--version":
		fmt.Printf("journal version %s\n", Version)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Println(`journal - A secure, encrypted journal with Git sync support

Usage:
  journal <command> [flags]

Available Commands:
  init              Initialize a new journal
  keygen            Generate a new age identity (key pair)
  add               Add a new journal entry
  list              List recent journal entries
  search            Search journal entries
  show              Show a specific journal entry
  delete            Delete a journal entry
  rebuild           Rebuild the search index from all entries
  list-journals     List all configured journals
  set-default       Set the default journal
  add-recipient     Add a recipient to a multi-recipient journal
  remove-recipient  Remove a recipient from a journal
  re-encrypt        Re-encrypt journal after changing recipients
  help              Show this help message
  version           Show version information

Global Flags:
  -j, --journal     Journal name to use (default: configured default journal)`)
}

// openJournal loads config and opens the specified (or default) journal
func openJournal(journalName string) (*entry.Journal, *config.Journal, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	var journalCfg *config.Journal
	if journalName == "" {
		journalCfg, err = cfg.GetDefaultJournal()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get default journal: %w\nHint: Use -j flag to specify a journal, or set a default with 'journal set-default <name>'", err)
		}
	} else {
		journalCfg, err = cfg.GetJournal(journalName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get journal: %w", err)
		}
	}

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open journal: %w", err)
	}

	return j, journalCfg, nil
}
