package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func runShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal show [entry-id] [flags]")
		fmt.Println("\nShow a specific journal entry")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: entry ID is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}

	j, _, err := openJournal(*journalName)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "%v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	ent, err := j.Get(fs.Arg(0))
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to get entry: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("ID: %s\n", ent.GetID()); err != nil {
		return 1
	}
	if _, err := fmt.Printf("Date: %s\n", ent.GetDate().Format("2006-01-02 15:04:05")); err != nil {
		return 1
	}
	if len(ent.GetTags()) > 0 {
		if _, err := fmt.Printf("Tags: %s\n", strings.Join(ent.GetTags(), ", ")); err != nil {
			return 1
		}
	}
	if _, err := fmt.Printf("\n%s\n", ent.GetContent()); err != nil {
		return 1
	}
	return 0
}

func runDelete(args []string) int {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal delete [entry-id] [flags]")
		fmt.Println("\nDelete a journal entry")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: entry ID is required\n\n"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}

	j, _, err := openJournal(*journalName)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "%v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if err := j.Delete(fs.Arg(0)); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to delete entry: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Entry %s deleted\n", fs.Arg(0)); err != nil {
		return 1
	}
	return 0
}

func runRebuild(args []string) int {
	fs := flag.NewFlagSet("rebuild", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal rebuild [flags]")
		fmt.Println("\nRebuild the search index from all entries")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	j, _, err := openJournal(*journalName)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "%v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Println("Rebuilding index..."); err != nil {
		return 1
	}
	if err := j.RebuildIndex(); err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to rebuild index: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Println("Index rebuilt successfully"); err != nil {
		return 1
	}
	return 0
}
