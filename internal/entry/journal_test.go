package entry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/crypto"
	"github.com/data-castle/journal/pkg/models"
)

func setupTestJournal(t *testing.T) (*Journal, *config.Journal) {
	t.Helper()
	tmpDir := t.TempDir()

	// Generate test key
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	// Setup SOPS_AGE_KEY_FILE
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

	// Create journal
	journalPath := filepath.Join(tmpDir, "test-journal")
	journalCfg := &config.Journal{
		Name: "test",
		Path: journalPath,
	}

	if err := InitializeJournal(journalCfg, []string{publicKey}); err != nil {
		t.Fatalf("failed to initialize journal: %v", err)
	}

	journal, err := NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to create journal: %v", err)
	}

	return journal, journalCfg
}

// mustAddEntry adds an entry and fails the test if there's an error
func mustAddEntry(t *testing.T, j *Journal, content string, tags []string) models.Entry {
	t.Helper()
	entry, err := j.Add(content, tags)
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}
	return entry
}

func TestInitializeJournal(t *testing.T) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	journalPath := filepath.Join(tmpDir, "test-journal")
	journalCfg := &config.Journal{
		Name: "test",
		Path: journalPath,
	}

	err = InitializeJournal(journalCfg, []string{publicKey})
	if err != nil {
		t.Fatalf("InitializeJournal failed: %v", err)
	}

	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		t.Error("journal directory was not created")
	}

	sopsPath := filepath.Join(journalPath, ".sops.yaml")
	if _, err := os.Stat(sopsPath); os.IsNotExist(err) {
		t.Error(".sops.yaml was not created")
	}

	indexPath := filepath.Join(journalPath, "index.yaml")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index.yaml was not created")
	}
}

func TestJournalAdd(t *testing.T) {
	journal, _ := setupTestJournal(t)

	entry, err := journal.Add("Test content", []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}

	if entry.GetContent() != "Test content" {
		t.Errorf("expected content 'Test content', got '%s'", entry.GetContent())
	}

	tags := entry.GetTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	if len(journal.index.Entries) != 1 {
		t.Errorf("expected 1 entry in index, got %d", len(journal.index.Entries))
	}
}

func TestJournalGet(t *testing.T) {
	journal, _ := setupTestJournal(t)

	addedEntry := mustAddEntry(t, journal, "Test content", []string{"tag1"})
	entryID := addedEntry.GetID()

	retrievedEntry, err := journal.Get(entryID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrievedEntry.GetID() != entryID {
		t.Errorf("expected ID %s, got %s", entryID, retrievedEntry.GetID())
	}

	if retrievedEntry.GetContent() != "Test content" {
		t.Errorf("expected content 'Test content', got '%s'", retrievedEntry.GetContent())
	}
}

func TestJournalGet_NotFound(t *testing.T) {
	journal, _ := setupTestJournal(t)

	_, err := journal.Get("nonexistent-id")
	if err == nil {
		t.Fatal("expected error when getting nonexistent entry")
	}
}

func TestJournalDelete(t *testing.T) {
	journal, _ := setupTestJournal(t)

	entry := mustAddEntry(t, journal, "Entry to delete", []string{})
	entryID := entry.GetID()

	err := journal.Delete(entryID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if len(journal.index.Entries) != 0 {
		t.Errorf("expected 0 entries in index, got %d", len(journal.index.Entries))
	}

	_, err = journal.Get(entryID)
	if err == nil {
		t.Fatal("expected error when getting deleted entry")
	}
}

func TestJournalUpdate(t *testing.T) {
	journal, _ := setupTestJournal(t)

	entry := mustAddEntry(t, journal, "Original content", []string{"tag1"})
	entryID := entry.GetID()

	updatedEntry, err := journal.Update(entryID, "Updated content", []string{"tag2", "tag3"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updatedEntry.GetContent() != "Updated content" {
		t.Errorf("expected content 'Updated content', got '%s'", updatedEntry.GetContent())
	}

	tags := updatedEntry.GetTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	retrievedEntry, err := journal.Get(entryID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrievedEntry.GetContent() != "Updated content" {
		t.Errorf("expected persisted content 'Updated content', got '%s'", retrievedEntry.GetContent())
	}
}

func TestJournalSearchByDate(t *testing.T) {
	journal, _ := setupTestJournal(t)

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	mustAddEntry(t, journal, "Entry today", []string{})

	entries, err := journal.SearchByDate(today)
	if err != nil {
		t.Fatalf("SearchByDate failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry for today, got %d", len(entries))
	}

	entries, err = journal.SearchByDate(yesterday)
	if err != nil {
		t.Fatalf("SearchByDate failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries for yesterday, got %d", len(entries))
	}
}

func TestJournalSearchByDateRange(t *testing.T) {
	journal, _ := setupTestJournal(t)

	mustAddEntry(t, journal, "Entry 1", []string{})
	mustAddEntry(t, journal, "Entry 2", []string{})

	start := time.Now().AddDate(0, 0, -1)
	end := time.Now().AddDate(0, 0, 1)

	entries, err := journal.SearchByDateRange(start, end)
	if err != nil {
		t.Fatalf("SearchByDateRange failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestJournalSearchByTag(t *testing.T) {
	journal, _ := setupTestJournal(t)

	mustAddEntry(t, journal, "Entry 1", []string{"work"})
	mustAddEntry(t, journal, "Entry 2", []string{"personal"})
	mustAddEntry(t, journal, "Entry 3", []string{"work", "important"})

	entries, err := journal.SearchByTag("work")
	if err != nil {
		t.Fatalf("SearchByTag failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries with tag 'work', got %d", len(entries))
	}
}

func TestJournalSearchByTags(t *testing.T) {
	journal, _ := setupTestJournal(t)

	mustAddEntry(t, journal, "Entry 1", []string{"work", "important"})
	mustAddEntry(t, journal, "Entry 2", []string{"work"})
	mustAddEntry(t, journal, "Entry 3", []string{"important"})

	entries, err := journal.SearchByTags([]string{"work", "important"})
	if err != nil {
		t.Fatalf("SearchByTags failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry with both tags, got %d", len(entries))
	}
}

func TestJournalListRecent(t *testing.T) {
	journal, _ := setupTestJournal(t)

	for i := 1; i <= 5; i++ {
		mustAddEntry(t, journal, "Entry", []string{})
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	entries, err := journal.ListRecent(3)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Verify entries are sorted by date descending
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].GetDate().Before(entries[i+1].GetDate()) {
			t.Error("entries are not sorted by date descending")
		}
	}
}

func TestJournalListAll(t *testing.T) {
	journal, _ := setupTestJournal(t)

	for i := 1; i <= 3; i++ {
		mustAddEntry(t, journal, "Entry", []string{})
	}

	metas := journal.ListAll()

	if len(metas) != 3 {
		t.Errorf("expected 3 metadata entries, got %d", len(metas))
	}
}

func TestJournalRebuildIndex(t *testing.T) {
	journal, journalCfg := setupTestJournal(t)

	mustAddEntry(t, journal, "Entry 1", []string{"tag1"})
	mustAddEntry(t, journal, "Entry 2", []string{"tag2"})

	err := journal.RebuildIndex()
	if err != nil {
		t.Fatalf("RebuildIndex failed: %v", err)
	}

	if len(journal.index.Entries) != 2 {
		t.Errorf("expected 2 entries in rebuilt index, got %d", len(journal.index.Entries))
	}

	journal2, err := NewJournalFromConfig(journalCfg)
	if err != nil {
		t.Fatalf("failed to create new journal: %v", err)
	}

	if len(journal2.index.Entries) != 2 {
		t.Errorf("expected 2 entries in reloaded index, got %d", len(journal2.index.Entries))
	}
}

func TestJournalAddRecipient(t *testing.T) {
	journal, _ := setupTestJournal(t)

	identity2, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey2 := identity2.Recipient().String()

	err = journal.AddRecipient(publicKey2)
	if err != nil {
		t.Fatalf("AddRecipient failed: %v", err)
	}

	recipients, err := crypto.ReadSOPSConfig(journal.config.Path)
	if err != nil {
		t.Fatalf("ReadSOPSConfig failed: %v", err)
	}

	if len(recipients) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(recipients))
	}
}

func TestJournalRemoveRecipient(t *testing.T) {
	journal, journalCfg := setupTestJournal(t)

	identity2, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey2 := identity2.Recipient().String()

	err = journal.AddRecipient(publicKey2)
	if err != nil {
		t.Fatalf("AddRecipient failed: %v", err)
	}

	err = journal.RemoveRecipient(publicKey2)
	if err != nil {
		t.Fatalf("RemoveRecipient failed: %v", err)
	}

	recipients, err := crypto.ReadSOPSConfig(journalCfg.Path)
	if err != nil {
		t.Fatalf("ReadSOPSConfig failed: %v", err)
	}

	if len(recipients) != 1 {
		t.Errorf("expected 1 recipient, got %d", len(recipients))
	}
}

func TestJournalListRecipients(t *testing.T) {
	journal, _ := setupTestJournal(t)

	recipients, err := journal.ListRecipients()
	if err != nil {
		t.Fatalf("ListRecipients failed: %v", err)
	}

	if len(recipients) != 1 {
		t.Errorf("expected 1 recipient, got %d", len(recipients))
	}
}

func TestJournalReEncrypt(t *testing.T) {
	journal, _ := setupTestJournal(t)

	entry1 := mustAddEntry(t, journal, "Entry 1", []string{})
	mustAddEntry(t, journal, "Entry 2", []string{})

	err := journal.ReEncrypt()
	if err != nil {
		t.Fatalf("ReEncrypt failed: %v", err)
	}

	retrievedEntry, err := journal.Get(entry1.GetID())
	if err != nil {
		t.Fatalf("Get failed after re-encrypt: %v", err)
	}

	if retrievedEntry.GetContent() != "Entry 1" {
		t.Errorf("expected content 'Entry 1', got '%s'", retrievedEntry.GetContent())
	}
}

// Tests for prefix matching functionality

func TestGetWithPrefixMatching(t *testing.T) {
	journal, _ := setupTestJournal(t)

	entry := mustAddEntry(t, journal, "Test entry", []string{"tag1"})
	fullID := entry.GetID()

	tests := []struct {
		name      string
		id        string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "Full ID",
			id:        fullID,
			shouldErr: false,
		},
		{
			name:      "8 character prefix",
			id:        fullID[:8],
			shouldErr: false,
		},
		{
			name:      "12 character prefix",
			id:        fullID[:12],
			shouldErr: false,
		},
		{
			name:      "16 character prefix",
			id:        fullID[:16],
			shouldErr: false,
		},
		{
			name:      "Less than 8 characters",
			id:        fullID[:7],
			shouldErr: true,
			errMsg:    "entry ID must be at least 8 characters",
		},
		{
			name:      "Non-existent prefix",
			id:        "ffffffff",
			shouldErr: true,
			errMsg:    "entry not found",
		},
		{
			name:      "Empty string",
			id:        "",
			shouldErr: true,
			errMsg:    "entry ID must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := journal.Get(tt.id)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("Expected error message containing %q, got %q", tt.errMsg, err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if retrieved.GetID() != fullID {
				t.Errorf("Expected ID %s, got %s", fullID, retrieved.GetID())
			}

			if retrieved.GetContent() != "Test entry" {
				t.Errorf("Expected content 'Test entry', got %q", retrieved.GetContent())
			}
		})
	}
}

func TestDeleteWithPrefixMatching(t *testing.T) {
	journal, _ := setupTestJournal(t)

	entry1 := mustAddEntry(t, journal, "Entry to delete", []string{})
	fullID1 := entry1.GetID()
	shortID1 := fullID1[:8]

	entry2 := mustAddEntry(t, journal, "Entry to keep", []string{})
	fullID2 := entry2.GetID()

	tests := []struct {
		name      string
		id        string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "Delete with 8 char prefix",
			id:        shortID1,
			shouldErr: false,
		},
		{
			name:      "Delete with full ID",
			id:        fullID2,
			shouldErr: false,
		},
		{
			name:      "Delete with short ID",
			id:        "1234567",
			shouldErr: true,
			errMsg:    "entry ID must be at least 8 characters",
		},
		{
			name:      "Delete non-existent",
			id:        "ffffffff",
			shouldErr: true,
			errMsg:    "entry not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := journal.Delete(tt.id)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}

	// Verify entries were actually deleted
	if _, err := journal.Get(fullID1); err == nil {
		t.Error("Entry 1 should have been deleted")
	}

	if _, err := journal.Get(fullID2); err == nil {
		t.Error("Entry 2 should have been deleted")
	}
}

func TestPrefixMatchingIntegration(t *testing.T) {
	journal, _ := setupTestJournal(t)

	// Add multiple entries
	entries := make([]models.Entry, 3)
	for i := 0; i < 3; i++ {
		entries[i] = mustAddEntry(t, journal, "Entry content", []string{})
		time.Sleep(time.Millisecond) // Ensure different UUIDs
	}

	t.Run("Get each entry by prefix", func(t *testing.T) {
		for i, entry := range entries {
			shortID := entry.GetID()[:8]
			retrieved, err := journal.Get(shortID)
			if err != nil {
				t.Errorf("Entry %d: Failed to get by prefix: %v", i, err)
				continue
			}
			if retrieved.GetID() != entry.GetID() {
				t.Errorf("Entry %d: Expected ID %s, got %s", i, entry.GetID(), retrieved.GetID())
			}
		}
	})

	t.Run("Delete by prefix", func(t *testing.T) {
		shortID := entries[0].GetID()[:8]
		if err := journal.Delete(shortID); err != nil {
			t.Fatalf("Failed to delete by prefix: %v", err)
		}

		// Verify deleted
		if _, err := journal.Get(entries[0].GetID()); err == nil {
			t.Error("Entry should have been deleted")
		}

		// Verify others still exist
		for i := 1; i < len(entries); i++ {
			if _, err := journal.Get(entries[i].GetID()); err != nil {
				t.Errorf("Entry %d should still exist: %v", i, err)
			}
		}
	})
}

func TestPrefixMatchingEdgeCases(t *testing.T) {
	journal, _ := setupTestJournal(t)

	t.Run("Empty journal", func(t *testing.T) {
		if _, err := journal.Get("12345678"); err == nil {
			t.Error("Expected error for empty journal")
		}

		if err := journal.Delete("12345678"); err == nil {
			t.Error("Expected error for empty journal")
		}
	})

	entry := mustAddEntry(t, journal, "Test", []string{})

	t.Run("Exact 8 character boundary", func(t *testing.T) {
		id8 := entry.GetID()[:8]
		retrieved, err := journal.Get(id8)
		if err != nil {
			t.Fatalf("Failed with 8 chars: %v", err)
		}
		if retrieved.GetID() != entry.GetID() {
			t.Error("Wrong entry retrieved")
		}
	})

	t.Run("7 characters fails", func(t *testing.T) {
		id7 := entry.GetID()[:7]
		if _, err := journal.Get(id7); err == nil {
			t.Error("Expected error with 7 characters")
		}
	})
}
