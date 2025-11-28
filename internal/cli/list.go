package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	count := fs.Int("count", 10, "Number of entries to show")
	fs.IntVar(count, "n", 10, "Number of entries to show (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal list [flags]")
		fmt.Println("\nList recent journal entries")
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

	metas := j.ListAll()

	if *count > 0 && *count < len(metas) {
		metas = metas[:*count]
	}

	if len(metas) == 0 {
		if _, err := fmt.Println("No entries found"); err != nil {
			return 1
		}
		return 0
	}

	for _, meta := range metas {
		if _, err := fmt.Printf("\n[%s] %s\n", meta.Date.Format("2006-01-02 15:04"), meta.Id[:8]); err != nil {
			return 1
		}
		if len(meta.Tags) > 0 {
			if _, err := fmt.Printf("Tags: %s\n", strings.Join(meta.Tags, ", ")); err != nil {
				return 1
			}
		}
	}
	return 0
}
