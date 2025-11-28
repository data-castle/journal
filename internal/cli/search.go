package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/data-castle/journal/pkg/models"
)

func runSearch(args []string) int {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	journalName := fs.String("journal", "", "Journal to use")
	fs.StringVar(journalName, "j", "", "Journal to use (shorthand)")
	onDate := fs.String("on", "", "Search entries on specific date (YYYY-MM-DD)")
	fromDate := fs.String("from", "", "Search entries from date (YYYY-MM-DD)")
	toDate := fs.String("to", "", "Search entries to date (YYYY-MM-DD)")
	tag := fs.String("tag", "", "Search entries with tag")
	tags := fs.String("tags", "", "Search entries with all tags (comma-separated)")
	lastDays := fs.Int("last", 0, "Search entries from last N days")
	fs.Usage = func() {
		fmt.Println("Usage: journal search [flags]")
		fmt.Println("\nSearch journal entries by date, date range, or tags")
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

	var entries []models.Entry
	var searchErr error

	switch {
	case *onDate != "":
		date, err := time.Parse("2006-01-02", *onDate)
		if err != nil {
			if _, ferr := fmt.Fprintf(os.Stderr, "Invalid date format: %v\n", err); ferr != nil {
				return 1
			}
			return 1
		}
		entries, searchErr = j.SearchByDate(date)

	case *fromDate != "" || *toDate != "":
		var start, end time.Time
		if *fromDate != "" {
			start, err = time.Parse("2006-01-02", *fromDate)
			if err != nil {
				if _, ferr := fmt.Fprintf(os.Stderr, "Invalid from date: %v\n", err); ferr != nil {
					return 1
				}
				return 1
			}
		}
		if *toDate != "" {
			end, err = time.Parse("2006-01-02", *toDate)
			if err != nil {
				if _, ferr := fmt.Fprintf(os.Stderr, "Invalid to date: %v\n", err); ferr != nil {
					return 1
				}
				return 1
			}
		} else {
			end = time.Now()
		}
		entries, searchErr = j.SearchByDateRange(start, end)

	case *lastDays > 0:
		end := time.Now()
		start := end.AddDate(0, 0, -*lastDays)
		entries, searchErr = j.SearchByDateRange(start, end)

	case *tag != "":
		entries, searchErr = j.SearchByTag(*tag)

	case *tags != "":
		tagList := strings.Split(*tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
		entries, searchErr = j.SearchByTags(tagList)

	default:
		if _, err := fmt.Println("Please specify search criteria"); err != nil {
			return 1
		}
		fs.Usage()
		return 1
	}

	if searchErr != nil {
		if _, ferr := fmt.Fprintf(os.Stderr, "Search failed: %v\n", searchErr); ferr != nil {
			return 1
		}
		return 1
	}

	if len(entries) == 0 {
		if _, err := fmt.Println("No entries found"); err != nil {
			return 1
		}
		return 0
	}

	if _, err := fmt.Printf("Found %d entries:\n", len(entries)); err != nil {
		return 1
	}
	for _, ent := range entries {
		if _, err := fmt.Printf("\n[%s] %s\n", ent.GetDate().Format("2006-01-02 15:04"), ent.GetID()[:8]); err != nil {
			return 1
		}
		if len(ent.GetTags()) > 0 {
			if _, err := fmt.Printf("Tags: %s\n", strings.Join(ent.GetTags(), ", ")); err != nil {
				return 1
			}
		}
		if _, err := fmt.Printf("%s\n", ent.GetContent()); err != nil {
			return 1
		}
	}
	return 0
}
