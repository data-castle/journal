package models

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// CurrentVersion is the latest version of the Entry model
	CurrentVersion = 1
)

// Entry is the interface that all entry versions must implement
type Entry interface {
	GetID() string
	GetDate() time.Time
	GetTags() []string
	GetFilePath() string
	GetContent() string
	GetVersion() int
	ToYaml() ([]byte, error)
}

// MetadataV1 contains the metadata for a journal entry (version 1)
type MetadataV1 struct {
	Version  int       `json:"version" yaml:"version"`
	Id       string    `json:"id" yaml:"id"`
	Date     time.Time `json:"date" yaml:"date"`
	Tags     []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	FilePath string    `json:"filepath" yaml:"filepath"`
}

// GetID returns the metadata ID
func (m *MetadataV1) GetID() string {
	return m.Id
}

// GetDate returns the metadata date
func (m *MetadataV1) GetDate() time.Time {
	return m.Date
}

// GetTags returns the metadata tags
func (m *MetadataV1) GetTags() []string {
	return m.Tags
}

// GetFilePath returns the metadata file path
func (m *MetadataV1) GetFilePath() string {
	return m.FilePath
}

// EntryV1 represents a journal entry (version 1)
type EntryV1 struct {
	MetadataV1 `json:",inline" yaml:",inline"`
	Content    string `json:"content" yaml:"content"`
}

// NewEntryV1 creates a new V1 entry with version set
func NewEntryV1(id string, date time.Time, content string, tags []string, filepath string) *EntryV1 {
	return &EntryV1{
		MetadataV1: MetadataV1{
			Version:  1,
			Id:       id,
			Date:     date,
			Tags:     tags,
			FilePath: filepath,
		},
		Content: content,
	}
}

// GetID returns the entry ID
func (e *EntryV1) GetID() string {
	return e.Id
}

// GetDate returns the entry date
func (e *EntryV1) GetDate() time.Time {
	return e.Date
}

// GetTags returns the entry tags
func (e *EntryV1) GetTags() []string {
	return e.Tags
}

// GetFilePath returns the file path
func (e *EntryV1) GetFilePath() string {
	return e.FilePath
}

// GetContent returns the entry content
func (e *EntryV1) GetContent() string {
	return e.Content
}

// GetVersion returns the version number
func (e *EntryV1) GetVersion() int {
	return e.Version
}

// ToYaml converts an EntryV1 to YAML format
func (e *EntryV1) ToYaml() ([]byte, error) {
	e.Version = 1
	return yaml.Marshal(e)
}

// versionDetector is used to peek at the version field
type versionDetector struct {
	Version int `yaml:"version"`
}

// ParseYaml parses YAML content into an Entry interface
func ParseYaml(content []byte) (Entry, error) {
	var detector versionDetector
	if err := yaml.Unmarshal(content, &detector); err != nil {
		return nil, fmt.Errorf("failed to detect version: %w", err)
	}

	switch detector.Version {
	case 1:
		var entry EntryV1
		if err := yaml.Unmarshal(content, &entry); err != nil {
			return nil, fmt.Errorf("failed to parse YAML as V1: %w", err)
		}
		if entry.Version != 1 {
			return nil, fmt.Errorf("failed to parse YAML as V1: invalid version: %d", entry.Version)
		}
		if entry.Id == "" {
			return nil, fmt.Errorf("entry ID is required")
		}
		if entry.Date.IsZero() {
			return nil, fmt.Errorf("entry date is required")
		}

		return &entry, nil

	default:
		return nil, fmt.Errorf("unsupported entry version: %d", detector.Version)
	}
}
