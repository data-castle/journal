package models

import (
	"testing"
	"time"
)

func TestIndexAddAndFind(t *testing.T) {
	idx := NewIndex()

	meta1 := &MetadataV1{
		Version:  1,
		Id:       "entry-1",
		Date:     time.Date(2024, 11, 19, 14, 0, 0, 0, time.UTC),
		Tags:     []string{"work", "meeting"},
		FilePath: "2024/11/entry-1.age",
	}

	meta2 := &MetadataV1{
		Version:  1,
		Id:       "entry-2",
		Date:     time.Date(2024, 11, 20, 10, 0, 0, 0, time.UTC),
		Tags:     []string{"personal"},
		FilePath: "2024/11/entry-2.age",
	}

	idx.Add(meta1)
	idx.Add(meta2)

	date1 := time.Date(2024, 11, 19, 0, 0, 0, 0, time.UTC)
	results := idx.FindByDate(date1)

	if len(results) != 1 {
		t.Errorf("Expected 1 result for date, got %d", len(results))
	}

	if results[0] != "entry-1" {
		t.Errorf("Expected entry-1, got %s", results[0])
	}

	workResults := idx.FindByTag("work")
	if len(workResults) != 1 {
		t.Errorf("Expected 1 result for 'work' tag, got %d", len(workResults))
	}

	bothTags := idx.FindByTags([]string{"work", "meeting"})
	if len(bothTags) != 1 {
		t.Errorf("Expected 1 result for both tags, got %d", len(bothTags))
	}

	start := time.Date(2024, 11, 19, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 11, 20, 23, 59, 59, 0, time.UTC)
	rangeResults := idx.FindByDateRange(start, end)

	if len(rangeResults) != 2 {
		t.Errorf("Expected 2 results for date range, got %d", len(rangeResults))
	}
}

func TestIndexRemove(t *testing.T) {
	idx := NewIndex()

	meta := &MetadataV1{
		Version:  1,
		Id:       "entry-1",
		Date:     time.Date(2024, 11, 19, 14, 0, 0, 0, time.UTC),
		Tags:     []string{"work"},
		FilePath: "2024/11/entry-1.age",
	}

	idx.Add(meta)

	if _, exists := idx.GetMetadata("entry-1"); !exists {
		t.Error("Entry should exist after adding")
	}

	idx.Remove("entry-1")
	if _, exists := idx.GetMetadata("entry-1"); exists {
		t.Error("Entry should not exist after removal")
	}

	// Verify it's removed from date index
	date := time.Date(2024, 11, 19, 0, 0, 0, 0, time.UTC)
	results := idx.FindByDate(date)
	if len(results) != 0 {
		t.Error("Entry should not be found by date after removal")
	}

	// Verify it's removed from tag index
	tagResults := idx.FindByTag("work")
	if len(tagResults) != 0 {
		t.Error("Entry should not be found by tag after removal")
	}
}

func TestIndexJSON(t *testing.T) {
	idx := NewIndex()

	meta := &MetadataV1{
		Version:  1,
		Id:       "entry-1",
		Date:     time.Date(2024, 11, 19, 14, 0, 0, 0, time.UTC),
		Tags:     []string{"work"},
		FilePath: "2024/11/entry-1.age",
	}

	idx.Add(meta)

	// Serialize to JSON
	jsonData, err := idx.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Deserialize from JSON
	newIdx, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	// Verify data is preserved
	newMeta, exists := newIdx.GetMetadata("entry-1")
	if !exists {
		t.Error("Entry should exist after JSON roundtrip")
	}

	if newMeta.Id != meta.Id {
		t.Error("ID should be preserved after JSON roundtrip")
	}

	if len(newMeta.Tags) != len(meta.Tags) {
		t.Error("Tags should be preserved after JSON roundtrip")
	}
}
