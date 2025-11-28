package cli

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/entry"
)

// setupTestJournal creates a test journal with encryption keys
// If tmpDir is empty, creates new tmpDir and config. Otherwise uses existing tmpDir and loads config.
// journalName: name of journal to create (defaults to "test" if empty)
func setupTestJournal(t *testing.T, tmpDir string, journalName string) (string, *config.Journal, string) {
	if journalName == "" {
		journalName = "test"
	}

	// Generate test key
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	var cfg *config.Config

	// Setup or load config
	if tmpDir == "" {
		tmpDir = t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		origFunc := config.GetConfigPathFunc
		config.GetConfigPathFunc = func() (string, error) {
			return configPath, nil
		}
		t.Cleanup(func() { config.GetConfigPathFunc = origFunc })

		cfg = config.NewConfig()
		if err := cfg.Save(); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	} else {
		cfg, err = config.LoadConfig()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
	}

	// Initialize journal
	journalPath := filepath.Join(tmpDir, journalName+"-journal")
	journalCfg := &config.Journal{
		Name: journalName,
		Path: journalPath,
	}

	if err := entry.InitializeJournal(journalCfg, []string{publicKey}); err != nil {
		t.Fatalf("failed to initialize journal: %v", err)
	}

	if err := cfg.AddJournal(journalCfg); err != nil {
		t.Fatalf("failed to add journal to config: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create SOPS_AGE_KEY_FILE
	keyPath := filepath.Join(tmpDir, "key.txt")
	keyContent := identity.String() + "\n"
	if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", keyPath); err != nil {
		t.Fatalf("failed to set SOPS_AGE_KEY_FILE: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	})

	return tmpDir, journalCfg, keyPath
}

// setupTestConfig creates a test config without initializing a journal
func setupTestConfig(t *testing.T) (string, string) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	origFunc := config.GetConfigPathFunc
	config.GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	t.Cleanup(func() { config.GetConfigPathFunc = origFunc })

	cfg := config.NewConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	return tmpDir, configPath
}
