package models

import (
	"strings"
	"testing"
	"time"
)

func TestEntryToYaml(t *testing.T) {
	entry := NewEntryV1(
		"test-id-123",
		time.Date(2024, 11, 19, 14, 30, 0, 0, time.UTC),
		"This is a test entry",
		[]string{"work", "meeting"},
		"",
	)

	yamlData, err := entry.ToYaml()
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}

	yamlStr := string(yamlData)
	if yamlStr == "" {
		t.Error("YAML should not be empty")
	}

	if !strings.Contains(yamlStr, "version: 1") {
		t.Error("YAML should contain version: 1")
	}

	if !strings.Contains(yamlStr, "test-id-123") {
		t.Error("YAML should contain entry ID")
	}

	if !strings.Contains(yamlStr, "work") {
		t.Error("YAML should contain tags")
	}

	if !strings.Contains(yamlStr, "This is a test entry") {
		t.Error("YAML should contain content")
	}
}

func TestParseYaml(t *testing.T) {
	yamlData := `version: 1
id: test-id-123
date: 2024-11-19T14:30:00Z
tags:
  - work
  - meeting
content: This is a test entry`

	entry, err := ParseYaml([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if entry.GetID() != "test-id-123" {
		t.Errorf("Expected ID 'test-id-123', got '%s'", entry.GetID())
	}

	if entry.GetContent() != "This is a test entry" {
		t.Errorf("Expected content 'This is a test entry', got '%s'", entry.GetContent())
	}

	if len(entry.GetTags()) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(entry.GetTags()))
	}

	expectedDate := time.Date(2024, 11, 19, 14, 30, 0, 0, time.UTC)
	if !entry.GetDate().Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, entry.GetDate())
	}

	if entry.GetVersion() != 1 {
		t.Errorf("Expected version 1, got %d", entry.GetVersion())
	}
}

func TestEntryToMetadata(t *testing.T) {
	entry := NewEntryV1(
		"test-id-123",
		time.Date(2024, 11, 19, 14, 30, 0, 0, time.UTC),
		"This is a test entry content",
		[]string{"work"},
		"",
	)
	if entry.Id != "test-id-123" {
		t.Error("Entry ID should be accessible via embedding")
	}
	if !entry.Date.Equal(time.Date(2024, 11, 19, 14, 30, 0, 0, time.UTC)) {
		t.Error("Entry date should be accessible via embedding")
	}
	if len(entry.Tags) != 1 {
		t.Error("Entry tags should be accessible via embedding")
	}
	meta := entry.MetadataV1
	if meta.Id != entry.Id {
		t.Error("Metadata ID should match entry ID")
	}
	if entry.Version != 1 {
		t.Error("Entry version should be 1")
	}
}
