package cli

import (
	"testing"
	"time"

	"github.com/data-castle/journal/internal/entry"
)

func TestRunSearch_ByDate(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	_, err = j.Add("Entry on specific date", []string{})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	args := []string{"-j", "test", "--on", today}
	exitCode := runSearch(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunSearch_ByTag(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	_, err = j.Add("Entry with tag1", []string{"tag1"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	args := []string{"-j", "test", "--tag", "tag1"}
	exitCode := runSearch(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunSearch_ByLastDays(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	_, err = j.Add("Recent entry", []string{})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	args := []string{"-j", "test", "--last", "7"}
	exitCode := runSearch(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunSearch_NoResults(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test", "--tag", "nonexistent"}
	exitCode := runSearch(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunSearch_NoCriteria(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test"}
	exitCode := runSearch(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing search criteria")
	}
}
