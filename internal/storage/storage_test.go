package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/data-castle/journal/internal/crypto"
	"github.com/data-castle/journal/pkg/models"
)

func setupTestStorage(t *testing.T) (*Storage, string) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	if err := crypto.CreateSOPSConfig(tmpDir, []string{publicKey}); err != nil {
		t.Fatalf("failed to create SOPS config: %v", err)
	}

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

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	return storage, tmpDir
}

func TestNewStorage(t *testing.T) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	if err := crypto.CreateSOPSConfig(tmpDir, []string{publicKey}); err != nil {
		t.Fatalf("failed to create SOPS config: %v", err)
	}

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Errorf("NewStorage failed: %v", err)
	}

	if storage == nil {
		t.Error("expected storage to be non-nil")
	}
}

func TestStorageInitialize(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)

	err := storage.Initialize()
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	entriesPath := filepath.Join(tmpDir, EntriesDir)
	if _, err := os.Stat(entriesPath); os.IsNotExist(err) {
		t.Error("entries directory was not created")
	}
}

func TestStorageInitialize_MissingSOPSConfig(t *testing.T) {
	tmpDir := t.TempDir()

	storage := &Storage{
		basePath: tmpDir,
	}

	err := storage.Initialize()
	if err == nil {
		t.Error("expected error when .sops.yaml is missing")
	}
}

func TestStorageSaveAndLoadEntry(t *testing.T) {
	storage, _ := setupTestStorage(t)

	if err := storage.Initialize(); err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	entryID := "test-entry-id"
	entryDate := time.Now()
	entry := models.NewEntryV1(entryID, entryDate, "Test content", []string{"tag1", "tag2"}, storage.GetEntryPath(entryDate, entryID))

	err := storage.SaveEntry(entry)
	if err != nil {
		t.Fatalf("SaveEntry failed: %v", err)
	}

	loadedEntry, err := storage.LoadEntry(entryID, entry.GetFilePath())
	if err != nil {
		t.Fatalf("LoadEntry failed: %v", err)
	}

	if loadedEntry.GetID() != entryID {
		t.Errorf("expected ID %s, got %s", entryID, loadedEntry.GetID())
	}

	if loadedEntry.GetContent() != "Test content" {
		t.Errorf("expected content 'Test content', got '%s'", loadedEntry.GetContent())
	}

	tags := loadedEntry.GetTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestStorageDeleteEntry(t *testing.T) {
	storage, _ := setupTestStorage(t)

	if err := storage.Initialize(); err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	entryID := "test-delete-id"
	entryDate := time.Now()
	entry := models.NewEntryV1(entryID, entryDate, "Entry to delete", []string{}, storage.GetEntryPath(entryDate, entryID))

	if err := storage.SaveEntry(entry); err != nil {
		t.Fatalf("SaveEntry failed: %v", err)
	}

	err := storage.DeleteEntry(entry.GetFilePath())
	if err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}

	_, err = storage.LoadEntry(entryID, entry.GetFilePath())
	if err == nil {
		t.Error("expected error when loading deleted entry")
	}
}

func TestStorageSaveAndLoadIndex(t *testing.T) {
	storage, _ := setupTestStorage(t)

	if err := storage.Initialize(); err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	index := models.NewIndex()
	metadata := models.Metadata{
		Id:       "test-id",
		Date:     time.Now(),
		Tags:     []string{"tag1"},
		FilePath: "2025/11/test-id.yaml",
	}
	index.Entries[metadata.Id] = metadata

	err := storage.SaveIndex(index)
	if err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	loadedIndex, err := storage.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if len(loadedIndex.Entries) != 1 {
		t.Errorf("expected 1 entry in index, got %d", len(loadedIndex.Entries))
	}

	loadedMeta, exists := loadedIndex.Entries["test-id"]
	if !exists {
		t.Fatal("expected to find entry 'test-id' in index")
	}

	if loadedMeta.Id != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", loadedMeta.Id)
	}
}

func TestStorageLoadIndex_Empty(t *testing.T) {
	storage, _ := setupTestStorage(t)

	if err := storage.Initialize(); err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	index, err := storage.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if index == nil {
		t.Error("expected non-nil index")
	}

	if len(index.Entries) != 0 {
		t.Errorf("expected empty index, got %d entries", len(index.Entries))
	}
}

func TestStorageListAllEntries(t *testing.T) {
	storage, _ := setupTestStorage(t)

	if err := storage.Initialize(); err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	dates := []time.Time{
		time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 20, 14, 0, 0, 0, time.UTC),
		time.Date(2025, 3, 10, 10, 0, 0, 0, time.UTC),
	}

	for i, date := range dates {
		entryID := filepath.Base(filepath.Dir(date.Format("test-id"))) + string(rune('0'+i))
		entry := models.NewEntryV1(entryID, date, "Content", []string{}, storage.GetEntryPath(date, entryID))
		if err := storage.SaveEntry(entry); err != nil {
			t.Fatalf("SaveEntry failed: %v", err)
		}
	}

	entries, err := storage.ListAllEntries()
	if err != nil {
		t.Fatalf("ListAllEntries failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestStorageGetEntryPath(t *testing.T) {
	storage, _ := setupTestStorage(t)

	date := time.Date(2025, 11, 26, 15, 30, 0, 0, time.UTC)
	entryID := "test-id-123"

	path := storage.GetEntryPath(date, entryID)

	expected := filepath.Join("2025", "11", "test-id-123.yaml")
	if path != expected {
		t.Errorf("expected path '%s', got '%s'", expected, path)
	}
}
