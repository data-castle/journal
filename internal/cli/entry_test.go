package cli

import (
	"testing"

	"github.com/data-castle/journal/internal/entry"
)

func TestRunShow_Success(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	ent, err := j.Add("Test entry content", []string{"tag1"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	entryID := ent.GetID()

	args := []string{"-j", "test", entryID}
	exitCode := runShow(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunShow_MissingID(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test"}
	exitCode := runShow(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing entry ID")
	}
}

func TestRunShow_InvalidID(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test", "invalid-id"}
	exitCode := runShow(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid entry ID")
	}
}

func TestRunDelete_Success(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	ent, err := j.Add("Test entry to delete", []string{})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	entryID := ent.GetID()

	args := []string{"-j", "test", entryID}
	exitCode := runDelete(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunDelete_MissingID(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test"}
	exitCode := runDelete(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing entry ID")
	}
}

func TestRunDelete_InvalidID(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test", "invalid-id"}
	exitCode := runDelete(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid entry ID")
	}
}

func TestRunRebuild_Success(t *testing.T) {
	_, journalCfg, _ := setupTestJournal(t, "", "")

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
	exitCode := runRebuild(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}
