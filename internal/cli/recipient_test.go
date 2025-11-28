package cli

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/crypto"
	"github.com/data-castle/journal/internal/entry"
)

func TestRunAddRecipient_WithAutoReencrypt(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate two test keys
	identity1, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity1: %v", err)
	}
	identity2, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity2: %v", err)
	}

	publicKey1 := identity1.Recipient().String()
	publicKey2 := identity2.Recipient().String()

	// Setup config
	configPath := filepath.Join(tmpDir, "config.yaml")
	origFunc := config.GetConfigPathFunc
	config.GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { config.GetConfigPathFunc = origFunc }()

	cfg := config.NewConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Initialize journal with first recipient
	journalPath := filepath.Join(tmpDir, "test-journal")
	journalCfg := &config.Journal{
		Name: "test",
		Path: journalPath,
	}

	if err := entry.InitializeJournal(journalCfg, []string{publicKey1}); err != nil {
		t.Fatalf("failed to initialize journal: %v", err)
	}

	if err := cfg.AddJournal(journalCfg); err != nil {
		t.Fatalf("failed to add journal to config: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create SOPS_AGE_KEY_FILE with first identity
	keyPath := filepath.Join(tmpDir, "key.txt")
	keyContent := identity1.String() + "\n"
	if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", keyPath); err != nil {
		t.Fatalf("failed to set SOPS_AGE_KEY_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	}()

	// Add some test entries
	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	ent, err := j.Add("Test entry 1", []string{"tag1"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}
	entryID := ent.GetID()

	// Verify .sops.yaml before adding recipient
	beforeRecipients, err := crypto.ReadSOPSConfig(journalPath)
	if err != nil {
		t.Fatalf("failed to read SOPS config before: %v", err)
	}
	if len(beforeRecipients) != 1 {
		t.Fatalf("expected 1 recipient before, got %d", len(beforeRecipients))
	}

	// Run add-recipient (should auto-reencrypt)
	args := []string{"-j", "test", publicKey2}
	exitCode := runAddRecipient(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify .sops.yaml after adding recipient
	afterRecipients, err := crypto.ReadSOPSConfig(journalPath)
	if err != nil {
		t.Fatalf("failed to read SOPS config after: %v", err)
	}
	if len(afterRecipients) != 2 {
		t.Fatalf("expected 2 recipients after, got %d", len(afterRecipients))
	}

	// Verify both recipients are present
	found1, found2 := false, false
	for _, r := range afterRecipients {
		if r == publicKey1 {
			found1 = true
		}
		if r == publicKey2 {
			found2 = true
		}
	}
	if !found1 {
		t.Error("original recipient not found after add")
	}
	if !found2 {
		t.Error("new recipient not found after add")
	}

	// Verify entries are still accessible with first key
	j, err = entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to reopen journal: %v", err)
	}

	retrievedEntry, err := j.Get(entryID)
	if err != nil {
		t.Fatalf("failed to get entry after reencrypt: %v", err)
	}

	if retrievedEntry.GetContent() != "Test entry 1" {
		t.Errorf("entry content mismatch: got %s", retrievedEntry.GetContent())
	}
}

func TestRunAddRecipient_MissingPublicKey(t *testing.T) {
	args := []string{"-j", "test"}
	exitCode := runAddRecipient(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing public key")
	}
}

func TestRunRemoveRecipient_WithAutoReencrypt(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate two test keys
	identity1, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity1: %v", err)
	}
	identity2, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity2: %v", err)
	}

	publicKey1 := identity1.Recipient().String()
	publicKey2 := identity2.Recipient().String()

	// Setup config
	configPath := filepath.Join(tmpDir, "config.yaml")
	origFunc := config.GetConfigPathFunc
	config.GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { config.GetConfigPathFunc = origFunc }()

	cfg := config.NewConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Initialize journal with both recipients
	journalPath := filepath.Join(tmpDir, "test-journal")
	journalCfg := &config.Journal{
		Name: "test",
		Path: journalPath,
	}

	if err := entry.InitializeJournal(journalCfg, []string{publicKey1, publicKey2}); err != nil {
		t.Fatalf("failed to initialize journal: %v", err)
	}

	if err := cfg.AddJournal(journalCfg); err != nil {
		t.Fatalf("failed to add journal to config: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create SOPS_AGE_KEY_FILE with first identity
	keyPath := filepath.Join(tmpDir, "key.txt")
	keyContent := identity1.String() + "\n"
	if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", keyPath); err != nil {
		t.Fatalf("failed to set SOPS_AGE_KEY_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	}()

	// Add test entry
	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	ent, err := j.Add("Test entry 1", []string{"tag1"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}
	entryID := ent.GetID()

	// Verify .sops.yaml before removing recipient
	beforeRecipients, err := crypto.ReadSOPSConfig(journalPath)
	if err != nil {
		t.Fatalf("failed to read SOPS config before: %v", err)
	}
	if len(beforeRecipients) != 2 {
		t.Fatalf("expected 2 recipients before, got %d", len(beforeRecipients))
	}

	// Run remove-recipient (should auto-reencrypt)
	args := []string{"-j", "test", publicKey2}
	exitCode := runRemoveRecipient(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify .sops.yaml after removing recipient
	afterRecipients, err := crypto.ReadSOPSConfig(journalPath)
	if err != nil {
		t.Fatalf("failed to read SOPS config after: %v", err)
	}
	if len(afterRecipients) != 1 {
		t.Fatalf("expected 1 recipient after, got %d", len(afterRecipients))
	}

	// Verify only first recipient remains
	if afterRecipients[0] != publicKey1 {
		t.Errorf("expected remaining recipient to be %s, got %s", publicKey1, afterRecipients[0])
	}

	// Verify entries are still accessible with first key
	j, err = entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to reopen journal: %v", err)
	}

	retrievedEntry, err := j.Get(entryID)
	if err != nil {
		t.Fatalf("failed to get entry after reencrypt: %v", err)
	}

	if retrievedEntry.GetContent() != "Test entry 1" {
		t.Errorf("entry content mismatch: got %s", retrievedEntry.GetContent())
	}
}

func TestRunRemoveRecipient_MissingPublicKey(t *testing.T) {
	args := []string{"-j", "test"}
	exitCode := runRemoveRecipient(args)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing public key")
	}
}

func TestRunReEncrypt(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test key
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	publicKey := identity.Recipient().String()

	// Setup config
	configPath := filepath.Join(tmpDir, "config.yaml")
	origFunc := config.GetConfigPathFunc
	config.GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { config.GetConfigPathFunc = origFunc }()

	cfg := config.NewConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Initialize journal
	journalPath := filepath.Join(tmpDir, "test-journal")
	journalCfg := &config.Journal{
		Name: "test",
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
	defer func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	}()

	// Add test entry
	j, err := entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}

	_, err = j.Add("Test entry", []string{"tag1"})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Run re-encrypt
	args := []string{"-j", "test"}
	exitCode := runReEncrypt(args)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify entries are still accessible
	j, err = entry.NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to reopen journal: %v", err)
	}

	entries, err := j.ListRecent(10)
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}
