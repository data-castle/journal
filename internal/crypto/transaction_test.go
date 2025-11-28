package crypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupAndRestoreSOPSConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a test .sops.yaml
	sopsPath := filepath.Join(tmpDir, ".sops.yaml")
	originalContent := "test: original"
	if err := os.WriteFile(sopsPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("failed to create test .sops.yaml: %v", err)
	}

	// Test backup
	backupPath, err := BackupSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("BackupSOPSConfig failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("backup file was not created: %s", backupPath)
	}

	// Verify backup has correct content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupContent) != originalContent {
		t.Errorf("backup content mismatch: got %q, want %q", string(backupContent), originalContent)
	}

	// Modify original file
	modifiedContent := "test: modified"
	if err := os.WriteFile(sopsPath, []byte(modifiedContent), 0600); err != nil {
		t.Fatalf("failed to modify .sops.yaml: %v", err)
	}

	// Test restore
	if err := RestoreSOPSConfig(tmpDir, backupPath); err != nil {
		t.Fatalf("RestoreSOPSConfig failed: %v", err)
	}

	// Verify original content is restored
	restoredContent, err := os.ReadFile(sopsPath)
	if err != nil {
		t.Fatalf("failed to read restored .sops.yaml: %v", err)
	}
	if string(restoredContent) != originalContent {
		t.Errorf("restored content mismatch: got %q, want %q", string(restoredContent), originalContent)
	}

	// Verify backup was removed
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Errorf("backup file still exists after restore: %s", backupPath)
	}
}

func TestRemoveBackup(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a test backup file
	backupPath := filepath.Join(tmpDir, ".sops.yaml.backup.test")
	if err := os.WriteFile(backupPath, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	// Test remove
	if err := RemoveBackup(backupPath); err != nil {
		t.Fatalf("RemoveBackup failed: %v", err)
	}

	// Verify backup was removed
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Errorf("backup file still exists after RemoveBackup: %s", backupPath)
	}
}

func TestTransactionalReEncrypt_Success(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Generate valid age recipients for testing
	recipients := generateRecipients(2)
	testRecipient := recipients[0]
	recipient2 := recipients[1]

	// Create a test .sops.yaml with first recipient
	if err := CreateSOPSConfig(tmpDir, []string{testRecipient}); err != nil {
		t.Fatalf("failed to create initial .sops.yaml: %v", err)
	}

	// Track which functions were called
	listCalled := false
	entryCount := 0
	indexCalled := false

	// Define test functions
	listEntriesFunc := func() ([]string, error) {
		listCalled = true
		return []string{"2024/01/entry1.yaml", "2024/01/entry2.yaml"}, nil
	}

	reEncryptEntryFunc := func(filePath string) error {
		entryCount++
		return nil // Success
	}

	reEncryptIndexFunc := func() error {
		indexCalled = true
		return nil // Success
	}

	// Add a second recipient
	newRecipients := []string{testRecipient, recipient2}

	// Execute transaction
	result, err := TransactionalReEncrypt(
		tmpDir,
		newRecipients,
		listEntriesFunc,
		reEncryptEntryFunc,
		reEncryptIndexFunc,
	)

	// Verify success
	if err != nil {
		t.Fatalf("TransactionalReEncrypt failed: %v", err)
	}

	// Verify all functions were called
	if !listCalled {
		t.Error("listEntriesFunc was not called")
	}
	if entryCount != 2 {
		t.Errorf("reEncryptEntryFunc called %d times, want 2", entryCount)
	}
	if !indexCalled {
		t.Error("reEncryptIndexFunc was not called")
	}

	// Verify result
	if result.TotalFiles != 2 {
		t.Errorf("TotalFiles = %d, want 2", result.TotalFiles)
	}
	if result.SuccessfulFiles != 2 {
		t.Errorf("SuccessfulFiles = %d, want 2", result.SuccessfulFiles)
	}
	if len(result.FailedFiles) != 0 {
		t.Errorf("FailedFiles = %d, want 0", len(result.FailedFiles))
	}
	if !result.IndexSuccess {
		t.Error("IndexSuccess = false, want true")
	}

	// Verify .sops.yaml was updated
	readRecipients, err := ReadSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to read .sops.yaml after transaction: %v", err)
	}
	if len(readRecipients) != 2 {
		t.Errorf("recipients count = %d, want 2", len(readRecipients))
	}

	// Verify no backup files remain
	files, _ := os.ReadDir(tmpDir)
	for _, file := range files {
		if strings.Contains(file.Name(), ".backup.") {
			t.Errorf("backup file still exists: %s", file.Name())
		}
	}
}

func TestTransactionalReEncrypt_FailureRollback(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Generate valid age recipients for testing
	recipients := generateRecipients(2)
	testRecipient := recipients[0]
	recipient2 := recipients[1]

	// Create a test .sops.yaml with first recipient
	if err := CreateSOPSConfig(tmpDir, []string{testRecipient}); err != nil {
		t.Fatalf("failed to create initial .sops.yaml: %v", err)
	}

	// Read original recipients
	originalRecipients, err := ReadSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to read original recipients: %v", err)
	}

	// Define test functions that fail
	listEntriesFunc := func() ([]string, error) {
		return []string{"entry1.yaml"}, nil
	}

	reEncryptEntryFunc := func(filePath string) error {
		return os.ErrInvalid // Simulate failure
	}

	reEncryptIndexFunc := func() error {
		return nil
	}

	// Add a second recipient
	newRecipients := []string{testRecipient, recipient2}

	// Execute transaction (should fail and rollback)
	result, err := TransactionalReEncrypt(
		tmpDir,
		newRecipients,
		listEntriesFunc,
		reEncryptEntryFunc,
		reEncryptIndexFunc,
	)

	// Verify it failed
	if err == nil {
		t.Fatal("TransactionalReEncrypt should have failed but succeeded")
	}

	// Verify error message mentions rollback
	if !strings.Contains(err.Error(), "rolled back") {
		t.Errorf("error should mention rollback: %v", err)
	}

	// Verify result contains failure info
	if len(result.FailedFiles) != 1 {
		t.Errorf("FailedFiles = %d, want 1", len(result.FailedFiles))
	}

	// Verify .sops.yaml was rolled back to original
	currentRecipients, err := ReadSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to read .sops.yaml after rollback: %v", err)
	}

	if len(currentRecipients) != len(originalRecipients) {
		t.Errorf("recipients after rollback = %d, want %d", len(currentRecipients), len(originalRecipients))
	}

	if currentRecipients[0] != originalRecipients[0] {
		t.Errorf("recipient after rollback = %s, want %s", currentRecipients[0], originalRecipients[0])
	}

	// Verify no backup files remain
	files, _ := os.ReadDir(tmpDir)
	for _, file := range files {
		if strings.Contains(file.Name(), ".backup.") {
			t.Errorf("backup file still exists after rollback: %s", file.Name())
		}
	}
}

func TestReEncryptResult_FormatErrors(t *testing.T) {
	result := &ReEncryptResult{
		TotalFiles:      3,
		SuccessfulFiles: 1,
		FailedFiles: []FileError{
			{FilePath: "entry1.yaml", Error: os.ErrInvalid},
			{FilePath: "entry2.yaml", Error: os.ErrPermission},
		},
		IndexSuccess: false,
		IndexError:   os.ErrClosed,
	}

	formatted := result.FormatErrors()

	// Verify formatted output contains expected information
	if !strings.Contains(formatted, "Total files: 3") {
		t.Error("formatted output should contain total files")
	}
	if !strings.Contains(formatted, "Successful: 1") {
		t.Error("formatted output should contain successful count")
	}
	if !strings.Contains(formatted, "Failed: 2") {
		t.Error("formatted output should contain failed count")
	}
	if !strings.Contains(formatted, "entry1.yaml") {
		t.Error("formatted output should contain failed file path")
	}
	if !strings.Contains(formatted, "entry2.yaml") {
		t.Error("formatted output should contain failed file path")
	}
	if !strings.Contains(formatted, "Index encryption: FAILED") {
		t.Error("formatted output should indicate index failure")
	}
}

func TestPrepareAddRecipient(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Generate valid age recipients for testing
	recipients := generateRecipients(2)
	recipient1 := recipients[0]
	recipient2 := recipients[1]

	// Create a test .sops.yaml with one recipient
	if err := CreateSOPSConfig(tmpDir, []string{recipient1}); err != nil {
		t.Fatalf("failed to create .sops.yaml: %v", err)
	}

	// Test adding a new recipient
	newRecipients, err := PrepareAddRecipient(tmpDir, recipient2)
	if err != nil {
		t.Fatalf("PrepareAddRecipient failed: %v", err)
	}

	// Verify result
	if len(newRecipients) != 2 {
		t.Errorf("newRecipients count = %d, want 2", len(newRecipients))
	}
	if newRecipients[0] != recipient1 {
		t.Errorf("newRecipients[0] = %s, want %s", newRecipients[0], recipient1)
	}
	if newRecipients[1] != recipient2 {
		t.Errorf("newRecipients[1] = %s, want %s", newRecipients[1], recipient2)
	}

	// Test adding duplicate recipient (should fail)
	_, err = PrepareAddRecipient(tmpDir, recipient1)
	if err == nil {
		t.Error("PrepareAddRecipient should fail for duplicate recipient")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention duplicate: %v", err)
	}
}

func TestPrepareRemoveRecipient(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Generate valid age recipients for testing
	recipients := generateRecipients(3)
	recipient1 := recipients[0]
	recipient2 := recipients[1]
	nonExistentRecipient := recipients[2]

	// Create a test .sops.yaml with two recipients
	if err := CreateSOPSConfig(tmpDir, []string{recipient1, recipient2}); err != nil {
		t.Fatalf("failed to create .sops.yaml: %v", err)
	}

	// Test removing a recipient
	newRecipients, err := PrepareRemoveRecipient(tmpDir, recipient2)
	if err != nil {
		t.Fatalf("PrepareRemoveRecipient failed: %v", err)
	}

	// Verify result
	if len(newRecipients) != 1 {
		t.Errorf("newRecipients count = %d, want 1", len(newRecipients))
	}
	if newRecipients[0] != recipient1 {
		t.Errorf("newRecipients[0] = %s, want %s", newRecipients[0], recipient1)
	}

	// Test removing non-existent recipient (should fail)
	_, err = PrepareRemoveRecipient(tmpDir, nonExistentRecipient)
	if err == nil {
		t.Error("PrepareRemoveRecipient should fail for non-existent recipient")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}

	// Test removing last recipient (should fail)
	if err := CreateSOPSConfig(tmpDir, []string{recipient1}); err != nil {
		t.Fatalf("failed to create single-recipient .sops.yaml: %v", err)
	}

	_, err = PrepareRemoveRecipient(tmpDir, recipient1)
	if err == nil {
		t.Error("PrepareRemoveRecipient should fail when removing last recipient")
	}
	if !strings.Contains(err.Error(), "cannot remove last") {
		t.Errorf("error should mention cannot remove last: %v", err)
	}
}
