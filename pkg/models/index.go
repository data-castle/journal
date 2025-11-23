package models

import (
	"encoding/json"
	"time"
)

// IndexableMetadata represents metadata fields needed for indexing
type IndexableMetadata interface {
	GetID() string
	GetDate() time.Time
	GetTags() []string
	GetFilePath() string
}

// Metadata is the version-agnostic metadata stored in the index
type Metadata struct {
	Id       string    `json:"id" yaml:"id"`
	Date     time.Time `json:"date" yaml:"date"`
	Tags     []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	FilePath string    `json:"filepath" yaml:"filepath"`
}

// Index contains all entry metadata for fast searching
type Index struct {
	Version string              `json:"version"`
	Entries map[string]Metadata `json:"entries"` // ID -> metadata
	ByDate  map[string][]string `json:"by_date"` // date -> []ID
	ByTag   map[string][]string `json:"by_tag"`  // tag -> []ID
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{
		Version: "1.0",
		Entries: make(map[string]Metadata),
		ByDate:  make(map[string][]string),
		ByTag:   make(map[string][]string),
	}
}

// Add adds an entry to the index (accepts any IndexableMetadata)
func (idx *Index) Add(meta IndexableMetadata) {
	commonMeta := Metadata{
		Id:       meta.GetID(),
		Date:     meta.GetDate(),
		Tags:     meta.GetTags(),
		FilePath: meta.GetFilePath(),
	}

	idx.Entries[commonMeta.Id] = commonMeta

	dateKey := commonMeta.Date.Format("2006-01-02")
	idx.ByDate[dateKey] = appendUnique(idx.ByDate[dateKey], commonMeta.Id)

	for _, tag := range commonMeta.Tags {
		idx.ByTag[tag] = appendUnique(idx.ByTag[tag], commonMeta.Id)
	}
}

// Remove removes an entry from the index
func (idx *Index) Remove(id string) {
	meta, exists := idx.Entries[id]
	if !exists {
		return
	}

	delete(idx.Entries, id)

	dateKey := meta.Date.Format("2006-01-02")
	idx.ByDate[dateKey] = removeString(idx.ByDate[dateKey], id)
	if len(idx.ByDate[dateKey]) == 0 {
		delete(idx.ByDate, dateKey)
	}

	for _, tag := range meta.Tags {
		idx.ByTag[tag] = removeString(idx.ByTag[tag], id)
		if len(idx.ByTag[tag]) == 0 {
			delete(idx.ByTag, tag)
		}
	}
}

// FindByDate returns entry IDs for a specific date
func (idx *Index) FindByDate(date time.Time) []string {
	dateKey := date.Format("2006-01-02")
	return idx.ByDate[dateKey]
}

// FindByDateRange returns entry IDs within a date range
func (idx *Index) FindByDateRange(start, end time.Time) []string {
	var results []string
	seen := make(map[string]bool)

	// Iterate through each day in range
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		for _, id := range idx.ByDate[dateKey] {
			if !seen[id] {
				results = append(results, id)
				seen[id] = true
			}
		}
	}

	return results
}

// FindByTag returns entry IDs with a specific tag
func (idx *Index) FindByTag(tag string) []string {
	return idx.ByTag[tag]
}

// FindByTags returns entry IDs that have ALL specified tags (AND operation)
func (idx *Index) FindByTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	// Start with entries that have the first tag
	results := make(map[string]bool)
	for _, id := range idx.ByTag[tags[0]] {
		results[id] = true
	}

	// Filter by remaining tags
	for _, tag := range tags[1:] {
		tagIDs := make(map[string]bool)
		for _, id := range idx.ByTag[tag] {
			tagIDs[id] = true
		}

		for id := range results {
			if !tagIDs[id] {
				delete(results, id)
			}
		}
	}

	var ids []string
	for id := range results {
		ids = append(ids, id)
	}
	return ids
}

// GetMetadata returns metadata for a specific entry ID
func (idx *Index) GetMetadata(id string) (Metadata, bool) {
	meta, exists := idx.Entries[id]
	return meta, exists
}

// ToJSON serializes the index to JSON
func (idx *Index) ToJSON() ([]byte, error) {
	return json.MarshalIndent(idx, "", "  ")
}

// FromJSON deserializes the index from JSON
func FromJSON(data []byte) (*Index, error) {
	idx := &Index{}
	err := json.Unmarshal(data, idx)
	return idx, err
}

// Helper functions
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
