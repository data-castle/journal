package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/data-castle/journal/internal/crypto"
	"github.com/data-castle/journal/pkg/models"
)

const (
	IndexFileName = "index.yaml"
	EntriesDir    = "entries"
)

// Storage handles file system operations using SOPS encryption
type Storage struct {
	basePath  string
	encryptor *crypto.Encryptor
}

// NewStorage creates a new SOPS-based storage instance
func NewStorage(basePath string) (*Storage, error) {
	encryptor, err := crypto.NewEncryptor(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOPS encryptor: %w", err)
	}

	return &Storage{
		basePath:  basePath,
		encryptor: encryptor,
	}, nil
}

// NewStorageWithEncryptor creates a storage instance with an existing encryptor
// Useful for re-encryption scenarios where encryptor needs to be updated
func NewStorageWithEncryptor(basePath string, encryptor *crypto.Encryptor) *Storage {
	return &Storage{
		basePath:  basePath,
		encryptor: encryptor,
	}
}

// GetBasePath returns the base path of the storage
func (s *Storage) GetBasePath() string {
	return s.basePath
}

// Initialize creates the necessary directory structure and .sops.yaml if needed
func (s *Storage) Initialize() error {
	entriesPath := filepath.Join(s.basePath, EntriesDir)
	if err := os.MkdirAll(entriesPath, 0700); err != nil {
		return fmt.Errorf("failed to create entries directory: %w", err)
	}

	sopsConfigPath := filepath.Join(s.basePath, ".sops.yaml")
	if _, err := os.Stat(sopsConfigPath); os.IsNotExist(err) {
		return fmt.Errorf(".sops.yaml not found in %s - please initialize journal with recipients first", s.basePath)
	}

	return nil
}

// SaveEntry saves an entry to disk as encrypted YAML
func (s *Storage) SaveEntry(entry models.Entry) error {
	year := entry.GetDate().Format("2006")
	month := entry.GetDate().Format("01")

	dirPath := filepath.Join(s.basePath, EntriesDir, year, month)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filename := fmt.Sprintf("%s.yaml", entry.GetID())
	filePath := filepath.Join(dirPath, filename)

	if err := s.encryptor.EncryptYAMLInMemory(entry, filePath); err != nil {
		return fmt.Errorf("failed to encrypt and save entry: %w", err)
	}

	return nil
}

// LoadEntry loads an entry from disk
func (s *Storage) LoadEntry(id string, relFilePath string) (models.Entry, error) {
	fullPath := filepath.Join(s.basePath, EntriesDir, relFilePath)

	decryptedData, err := s.encryptor.DecryptFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt entry: %w", err)
	}

	entry, err := models.ParseYaml(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entry: %w", err)
	}

	return entry, nil
}

// DeleteEntry deletes an entry from disk
func (s *Storage) DeleteEntry(relFilePath string) error {
	fullPath := filepath.Join(s.basePath, EntriesDir, relFilePath)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete entry file: %w", err)
	}

	return nil
}

// SaveIndex saves the index to disk as encrypted YAML
func (s *Storage) SaveIndex(index *models.Index) error {
	indexPath := filepath.Join(s.basePath, IndexFileName)

	if err := s.encryptor.EncryptYAMLInMemory(index, indexPath); err != nil {
		return fmt.Errorf("failed to encrypt and save index: %w", err)
	}

	return nil
}

// LoadIndex loads the index from disk
func (s *Storage) LoadIndex() (*models.Index, error) {
	indexPath := filepath.Join(s.basePath, IndexFileName)

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Return new empty index
		return models.NewIndex(), nil
	}

	var index models.Index
	if err := s.encryptor.DecryptYAML(indexPath, &index); err != nil {
		return nil, fmt.Errorf("failed to decrypt and parse index: %w", err)
	}

	return &index, nil
}

// ListAllEntries recursively lists all entry files
func (s *Storage) ListAllEntries() ([]string, error) {
	var entries []string

	entriesPath := filepath.Join(s.basePath, EntriesDir)

	err := filepath.Walk(entriesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			relPath, err := filepath.Rel(entriesPath, path)
			if err != nil {
				return err
			}
			entries = append(entries, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	return entries, nil
}

// GetEntryPath returns the relative path for an entry file
func (s *Storage) GetEntryPath(date time.Time, id string) string {
	year := date.Format("2006")
	month := date.Format("01")
	filename := fmt.Sprintf("%s.yaml", id)
	return filepath.Join(year, month, filename)
}
