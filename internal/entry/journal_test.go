package entry

import (
	"os"
	"path/filepath"
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
