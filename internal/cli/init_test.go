package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/data-castle/journal/internal/config"
)

func TestRunInit_Success(t *testing.T) {
	tmpDir, _ := setupTestConfig(t)

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	journalPath := filepath.Join(tmpDir, "test-journal")
	args := []string{
		"--name", "test",
		"--path", journalPath,
		"--recipients", publicKey,
	}

	exitCode := runInit(args)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
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

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if _, exists := cfg.Journals["test"]; !exists {
		t.Error("journal was not added to config")
	}
}

func TestRunInit_MissingName(t *testing.T) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	args := []string{
		"--path", filepath.Join(tmpDir, "test-journal"),
		"--recipients", publicKey,
	}

	exitCode := runInit(args)
	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing name")
	}
}

func TestRunInit_MissingPath(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	publicKey := identity.Recipient().String()

	args := []string{
		"--name", "test",
		"--recipients", publicKey,
	}

	exitCode := runInit(args)
	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing path")
	}
}

func TestRunInit_MissingRecipients(t *testing.T) {
	tmpDir := t.TempDir()

	args := []string{
		"--name", "test",
		"--path", filepath.Join(tmpDir, "test-journal"),
	}

	exitCode := runInit(args)
	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing recipients")
	}
}

func TestRunInit_MultipleRecipients(t *testing.T) {
	tmpDir, _ := setupTestConfig(t)

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

	journalPath := filepath.Join(tmpDir, "test-journal")
	args := []string{
		"--name", "test",
		"--path", journalPath,
		"--recipients", publicKey1 + "," + publicKey2,
	}

	exitCode := runInit(args)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	sopsContent, err := os.ReadFile(filepath.Join(journalPath, ".sops.yaml"))
	if err != nil {
		t.Fatalf("failed to read .sops.yaml: %v", err)
	}

	content := string(sopsContent)
	if !strings.Contains(content, publicKey1) {
		t.Error("first recipient not found in .sops.yaml")
	}
	if !strings.Contains(content, publicKey2) {
		t.Error("second recipient not found in .sops.yaml")
	}
}
