package cli

import (
	"testing"

	"github.com/data-castle/journal/internal/entry"
)

func TestRunList_Success(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	// Add test entries for listing
	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	_, err = j.Add("Entry 1", []string{"tag1"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	_, err = j.Add("Entry 2", []string{"tag2"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	args := []string{"-j", "test"}
	exitCode := runList(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunList_WithCount(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	for i := 1; i <= 3; i++ {
		_, err = j.Add("Entry", []string{})
		if err != nil {
			t.Fatalf("failed to add entry: %v", err)
		}
	}

	args := []string{"-j", "test", "-n", "2"}
	exitCode := runList(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunList_Empty(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test"}
	exitCode := runList(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}
