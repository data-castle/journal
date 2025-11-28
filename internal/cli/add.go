package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func runAdd(args []string) int {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	tags := fs.String("tags", "", "Tags for the entry (comma-separated)")
	fs.StringVar(tags, "t", "", "Tags for the entry (shorthand)")
	fs.Usage = func() {
		fmt.Println("Usage: journal add [text] [flags]")
		fmt.Println("\nAdd a new journal entry")
		fmt.Println("\nFlags:")
		fs.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  journal add \"Today was great!\" -j personal")
		fmt.Println("  journal add \"Team meeting\" -j work -t meeting,notes")
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: entry text is required\n\n"); err != nil {
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

	content := strings.Join(fs.Args(), " ")

	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
	}

	ent, err := j.Add(content, tagList)
	if err != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Failed to add entry: %v\n", err); ferr != nil {
			return 1
		}
		return 1
	}

	if _, err := fmt.Printf("Entry added: %s\n", ent.GetID()[:8]); err != nil {
		return 1
	}
	if _, err := fmt.Printf("Date: %s\n", ent.GetDate().Format("2006-01-02 15:04:05")); err != nil {
		return 1
	}
	if len(tagList) > 0 {
		if _, err := fmt.Printf("Tags: %s\n", strings.Join(tagList, ", ")); err != nil {
			return 1
		}
	}
	return 0
}
