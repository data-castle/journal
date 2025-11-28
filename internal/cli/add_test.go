package cli

import (
	"testing"
)

func TestRunAdd_Success(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test", "Test entry content"}
	exitCode := runAdd(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunAdd_WithTags(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test", "-t", "tag1,tag2", "Test entry with tags"}
	exitCode := runAdd(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunAdd_MissingContent(t *testing.T) {
	setupTestJournal(t, "", "")

	args := []string{"-j", "test"}
	exitCode := runAdd(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing content")
	}
}
