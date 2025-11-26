package entry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/data-castle/journal/internal/config"
	"github.com/data-castle/journal/internal/crypto"
	"github.com/data-castle/journal/internal/storage"
	"github.com/data-castle/journal/pkg/models"
	"github.com/google/uuid"
)

// Journal is the main entry point for journal operations using SOPS encryption
type Journal struct {
	config  *config.Journal
	storage *storage.Storage
	index   *models.Index
}

// NewJournalFromConfig creates a SOPS-based journal instance from config
func NewJournalFromConfig(cfg *config.Journal) (*Journal, error) {
	store, err := storage.NewStorage(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	if err := store.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	index, err := store.LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return &Journal{
		config:  cfg,
		storage: store,
		index:   index,
	}, nil
}

// InitializeJournal creates a new journal with specified recipients
func InitializeJournal(cfg *config.Journal, recipients []string) error {
	if err := os.MkdirAll(cfg.Path, 0700); err != nil {
		return fmt.Errorf("failed to create journal directory: %w", err)
	}

	if err := crypto.CreateSOPSConfig(cfg.Path, recipients); err != nil {
		return fmt.Errorf("failed to create SOPS config: %w", err)
	}

	store, err := storage.NewStorage(cfg.Path)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	if err := store.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	index := models.NewIndex()
	if err := store.SaveIndex(index); err != nil {
		return fmt.Errorf("failed to save initial index: %w", err)
	}

	return nil
}

// Add adds a new entry to the journal
func (j *Journal) Add(content string, tags []string) (models.Entry, error) {
	entry := models.NewEntryV1(
		uuid.New().String(),
		time.Now(),
		content,
		tags,
		"", // filepath will be determined by storage path
	)

	entry.FilePath = j.storage.GetEntryPath(entry.GetDate(), entry.GetID())

	if err := j.storage.SaveEntry(entry); err != nil {
		return nil, fmt.Errorf("failed to save entry: %w", err)
	}

	j.index.Add(&entry.MetadataV1)

	if err := j.storage.SaveIndex(j.index); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	return entry, nil
}

// Get retrieves a single entry by ID
func (j *Journal) Get(id string) (models.Entry, error) {
	meta, exists := j.index.GetMetadata(id)
	if !exists {
		return nil, fmt.Errorf("entry not found: %s", id)
	}

	entry, err := j.storage.LoadEntry(id, meta.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load entry: %w", err)
	}

	return entry, nil
}

// SearchByDate finds entries for a specific date
func (j *Journal) SearchByDate(date time.Time) ([]models.Entry, error) {
	ids := j.index.FindByDate(date)
	return j.loadEntries(ids)
}

// SearchByDateRange finds entries within a date range
func (j *Journal) SearchByDateRange(start, end time.Time) ([]models.Entry, error) {
	ids := j.index.FindByDateRange(start, end)
	return j.loadEntries(ids)
}

// SearchByTag finds entries with a specific tag
func (j *Journal) SearchByTag(tag string) ([]models.Entry, error) {
	ids := j.index.FindByTag(tag)
	return j.loadEntries(ids)
}

// SearchByTags finds entries with all specified tags (AND operation)
func (j *Journal) SearchByTags(tags []string) ([]models.Entry, error) {
	ids := j.index.FindByTags(tags)
	return j.loadEntries(ids)
}

// ListRecent lists the most recent N entries
func (j *Journal) ListRecent(count int) ([]models.Entry, error) {
	var metas []models.Metadata
	for _, meta := range j.index.Entries {
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Date.After(metas[j].Date)
	})

	if count > len(metas) {
		count = len(metas)
	}
	metas = metas[:count]

	// Load entries
	var entries []models.Entry
	var loadErrors []error
	for _, meta := range metas {
		entry, err := j.storage.LoadEntry(meta.Id, meta.FilePath)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to load entry %s: %w", meta.Id, err))
			continue
		}
		entries = append(entries, entry)
	}

	// Log warnings for failed entries to stderr
	if len(loadErrors) > 0 {
		for _, err := range loadErrors {
			if _, ferr := fmt.Fprintf(os.Stderr, "Warning: %v\n", err); ferr != nil {
				return nil, ferr
			}
		}
	}

	return entries, nil
}

// ListAll returns metadata for all entries (without loading full content)
func (j *Journal) ListAll() []models.Metadata {
	var metas []models.Metadata
	for _, meta := range j.index.Entries {
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Date.After(metas[j].Date)
	})

	return metas
}

// Delete removes an entry
func (j *Journal) Delete(id string) error {
	meta, exists := j.index.GetMetadata(id)
	if !exists {
		return fmt.Errorf("entry not found: %s", id)
	}

	if err := j.storage.DeleteEntry(meta.FilePath); err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	j.index.Remove(id)

	if err := j.storage.SaveIndex(j.index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// Update updates an existing entry
func (j *Journal) Update(id string, content string, tags []string) (models.Entry, error) {
	meta, exists := j.index.GetMetadata(id)
	if !exists {
		return nil, fmt.Errorf("entry not found: %s", id)
	}

	entry, err := j.storage.LoadEntry(id, meta.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load entry: %w", err)
	}

	// Type assert to V1 to update fields
	// Note: When adding new entry versions, add a type switch here to handle each version
	entryV1, ok := entry.(*models.EntryV1)
	if !ok {
		return nil, fmt.Errorf("unsupported entry version for update")
	}

	entryV1.Content = content
	entryV1.Tags = tags

	if err := j.storage.SaveEntry(entryV1); err != nil {
		return nil, fmt.Errorf("failed to save entry: %w", err)
	}

	// Update index
	j.index.Remove(id)
	j.index.Add(&entryV1.MetadataV1)

	if err := j.storage.SaveIndex(j.index); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	return entryV1, nil
}

// RebuildIndex rebuilds the index from all entry files
func (j *Journal) RebuildIndex() error {
	newIndex := models.NewIndex()

	files, err := j.storage.ListAllEntries()
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	// Load each entry and add to index
	for _, relFilePath := range files {
		filename := filepath.Base(relFilePath)
		id := filename[:len(filename)-len(".yaml")]

		entry, err := j.storage.LoadEntry(id, relFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load entry %s: %v\n", relFilePath, err)
			continue
		}

		newIndex.Add(entry)
	}

	j.index = newIndex

	if err := j.storage.SaveIndex(j.index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// ReEncrypt re-encrypts all entries and index with updated recipients
// This is useful when adding/removing recipients in .sops.yaml
func (j *Journal) ReEncrypt() error {
	files, err := j.storage.ListAllEntries()
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	// Re-encrypt each entry by loading and saving
	for _, relFilePath := range files {
		filename := filepath.Base(relFilePath)
		id := filename[:len(filename)-len(".yaml")]

		entry, err := j.storage.LoadEntry(id, relFilePath)
		if err != nil {
			return fmt.Errorf("failed to load entry %s: %w", relFilePath, err)
		}

		if err := j.storage.SaveEntry(entry); err != nil {
			return fmt.Errorf("failed to re-encrypt entry %s: %w", relFilePath, err)
		}
	}

	// Re-encrypt index
	if err := j.storage.SaveIndex(j.index); err != nil {
		return fmt.Errorf("failed to re-encrypt index: %w", err)
	}

	return nil
}

// Helper function to load multiple entries
func (j *Journal) loadEntries(ids []string) ([]models.Entry, error) {
	var entries []models.Entry
	var loadErrors []error

	for _, id := range ids {
		meta, exists := j.index.GetMetadata(id)
		if !exists {
			continue
		}

		entry, err := j.storage.LoadEntry(id, meta.FilePath)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to load entry %s: %w", id, err))
			continue
		}

		entries = append(entries, entry)
	}

	// Log warnings for failed entries to stderr
	if len(loadErrors) > 0 {
		for _, err := range loadErrors {
			if _, ferr := fmt.Fprintf(os.Stderr, "Warning: %v\n", err); ferr != nil {
				return nil, ferr
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].GetDate().After(entries[j].GetDate())
	})

	return entries, nil
}

// AddRecipient adds a new recipient to the journal's .sops.yaml
func (j *Journal) AddRecipient(publicKey string) error {
	if err := crypto.AddRecipient(j.config.Path, publicKey); err != nil {
		return fmt.Errorf("failed to add recipient: %w", err)
	}
	return nil
}

// RemoveRecipient removes a recipient from the journal's .sops.yaml
func (j *Journal) RemoveRecipient(publicKey string) error {
	if err := crypto.RemoveRecipient(j.config.Path, publicKey); err != nil {
		return fmt.Errorf("failed to remove recipient: %w", err)
	}
	return nil
}

// ListRecipients returns all recipients from the journal's .sops.yaml
func (j *Journal) ListRecipients() ([]string, error) {
	recipients, err := crypto.ReadSOPSConfig(j.config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipients: %w", err)
	}
	return recipients, nil
}
