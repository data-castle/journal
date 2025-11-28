package cli

import (
	"testing"

	"github.com/data-castle/journal/internal/config"
)

func TestRunListJournals_Empty(t *testing.T) {
	setupTestConfig(t)

	args := []string{}
	exitCode := runListJournals(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunListJournals_WithJournals(t *testing.T) {
	tmpDir, _, _ := setupTestJournal(t, "", "journal1")
	setupTestJournal(t, tmpDir, "journal2")

	args := []string{}
	exitCode := runListJournals(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunSetDefault_Success(t *testing.T) {
	tmpDir, _, _ := setupTestJournal(t, "", "journal1")
	setupTestJournal(t, tmpDir, "journal2")

	args := []string{"journal2"}
	exitCode := runSetDefault(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if cfg.DefaultJournal != "journal2" {
		t.Errorf("expected default journal to be 'journal2', got '%s'", cfg.DefaultJournal)
	}
}

func TestRunSetDefault_MissingName(t *testing.T) {
	setupTestConfig(t)

	args := []string{}
	exitCode := runSetDefault(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing name")
	}
}

func TestRunSetDefault_InvalidName(t *testing.T) {
	setupTestConfig(t)

	args := []string{"nonexistent"}
	exitCode := runSetDefault(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid journal name")
	}
}
