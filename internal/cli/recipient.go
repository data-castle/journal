package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/data-castle/journal/internal/crypto"
)

func runAddRecipient(args []string) int {
	fs := flag.NewFlagSet("add-recipient", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal add-recipient <public-key> [flags]")
		fmt.Println("\nAdd a recipient to a journal")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: recipient public key is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}

	recipient := fs.Arg(0)

	j, journalCfg, err := openJournal(*journalName)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "%v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	newRecipients, err := crypto.PrepareAddRecipient(journalCfg.Path, recipient)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to prepare recipient addition: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Adding recipient to journal '%s'\n", journalCfg.Name); err != nil {
		return 1
	}
	if _, err := fmt.Println("Re-encrypting all entries with new recipient..."); err != nil {
		return 1
	}

	if err := j.ReEncryptWithRecipients(newRecipients); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to add recipient: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Println("Re-encryption complete"); err != nil {
		return 1
	}
	if _, err := fmt.Printf("Successfully added recipient to journal '%s'\n", journalCfg.Name); err != nil {
		return 1
	}
	return 0
}

func runRemoveRecipient(args []string) int {
	fs := flag.NewFlagSet("remove-recipient", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal remove-recipient <public-key> [flags]")
		fmt.Println("\nRemove a recipient from a journal")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: recipient public key is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}

	recipient := fs.Arg(0)

	j, journalCfg, err := openJournal(*journalName)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "%v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	newRecipients, err := crypto.PrepareRemoveRecipient(journalCfg.Path, recipient)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to prepare recipient removal: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Removing recipient from journal '%s'\n", journalCfg.Name); err != nil {
		return 1
	}
	if _, err := fmt.Println("Re-encrypting all entries without removed recipient..."); err != nil {
		return 1
	}

	if err := j.ReEncryptWithRecipients(newRecipients); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to remove recipient: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Println("Re-encryption complete"); err != nil {
		return 1
	}
	if _, err := fmt.Printf("Successfully removed recipient from journal '%s'\n", journalCfg.Name); err != nil {
		return 1
	}
	return 0
}

func runReEncrypt(args []string) int {
	fs := flag.NewFlagSet("re-encrypt", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal re-encrypt [flags]")
		fmt.Println("\nRe-encrypt all entries with current recipient list from .sops.yaml")
		fmt.Println("Use this after adding or removing recipients")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	j, journalCfg, err := openJournal(*journalName)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "%v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Println("Re-encrypting all entries..."); err != nil {
		return 1
	}
	if err := j.ReEncrypt(); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to re-encrypt: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Re-encryption complete for journal '%s'\n", journalCfg.Name); err != nil {
		return 1
	}
	return 0
}
