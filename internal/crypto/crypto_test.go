package crypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestNewEncryptor(t *testing.T) {
	tmpDir := t.TempDir()

	recipients := generateRecipients(1)

	err := CreateSOPSConfig(tmpDir, recipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	enc, err := NewEncryptor(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	if enc == nil {
		t.Fatal("expected non-nil encryptor")
	}

	if enc.journalPath != tmpDir {
		t.Errorf("expected journalPath %s, got %s", tmpDir, enc.journalPath)
	}

	if len(enc.recipients) != len(recipients) {
		t.Errorf("expected %d recipients, got %d", len(recipients), len(enc.recipients))
	}
}

func TestNewEncryptor_MissingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := NewEncryptor(tmpDir)
	if err == nil {
		t.Error("expected error when .sops.yaml is missing")
	}

	if !strings.Contains(err.Error(), "failed to read SOPS config") {
		t.Errorf("expected 'failed to read SOPS config' error, got: %v", err)
	}
}

func TestCreateSOPSConfig(t *testing.T) {
	tmpDir := t.TempDir()

	recipients := generateRecipients(2)

	err := CreateSOPSConfig(tmpDir, recipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".sops.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal(".sops.yaml file was not created")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .sops.yaml: %v", err)
	}

	content := string(data)
	for _, recipient := range recipients {
		if !strings.Contains(content, recipient) {
			t.Errorf("recipient %s not found in .sops.yaml", recipient)
		}
	}

	if !strings.Contains(content, "index\\.yaml$") {
		t.Error("index.yaml rule not found in .sops.yaml")
	}

	if !strings.Contains(content, "entries/.*\\.yaml$") {
		t.Error("entries rule not found in .sops.yaml")
	}
}

func TestCreateSOPSConfig_NoRecipients(t *testing.T) {
	tmpDir := t.TempDir()

	err := CreateSOPSConfig(tmpDir, []string{})
	if err == nil {
		t.Error("expected error when creating config with no recipients")
	}

	if !strings.Contains(err.Error(), "no recipients provided") {
		t.Errorf("expected 'no recipients provided' error, got: %v", err)
	}
}

func TestReadSOPSConfig(t *testing.T) {
	tmpDir := t.TempDir()

	expectedRecipients := generateRecipients(2)

	err := CreateSOPSConfig(tmpDir, expectedRecipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	recipients, err := ReadSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("ReadSOPSConfig failed: %v", err)
	}

	if len(recipients) != len(expectedRecipients) {
		t.Fatalf("expected %d recipients, got %d", len(expectedRecipients), len(recipients))
	}

	for i, r := range recipients {
		if r != expectedRecipients[i] {
			t.Errorf("expected recipient %s, got %s", expectedRecipients[i], r)
		}
	}
}

func TestReadSOPSConfig_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ReadSOPSConfig(tmpDir)
	if err == nil {
		t.Error("expected error when reading missing .sops.yaml")
	}

	if !strings.Contains(err.Error(), "failed to read .sops.yaml") {
		t.Errorf("expected 'failed to read .sops.yaml' error, got: %v", err)
	}
}

func TestReadSOPSConfig_EmptyCreationRules(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, ".sops.yaml")
	emptyConfig := "creation_rules: []\n"
	err := os.WriteFile(configPath, []byte(emptyConfig), 0644)
	if err != nil {
		t.Fatalf("failed to write empty config: %v", err)
	}

	_, err = ReadSOPSConfig(tmpDir)
	if err == nil {
		t.Error("expected error when reading config with no creation rules")
	}

	if !strings.Contains(err.Error(), "no creation rules found") {
		t.Errorf("expected 'no creation rules found' error, got: %v", err)
	}
}

func TestReadSOPSConfig_NoAgeRecipients(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, ".sops.yaml")
	configWithoutAge := "creation_rules:\n  - path_regex: test\n"
	err := os.WriteFile(configPath, []byte(configWithoutAge), 0644)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err = ReadSOPSConfig(tmpDir)
	if err == nil {
		t.Error("expected error when reading config with no age recipients")
	}

	if !strings.Contains(err.Error(), "no age recipients found") {
		t.Errorf("expected 'no age recipients found' error, got: %v", err)
	}
}

func TestAddRecipient(t *testing.T) {
	tmpDir := t.TempDir()

	initialRecipients := generateRecipients(1)

	err := CreateSOPSConfig(tmpDir, initialRecipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	newRecipient := generateRecipients(1)[0]

	err = AddRecipient(tmpDir, newRecipient)
	if err != nil {
		t.Fatalf("AddRecipient failed: %v", err)
	}

	recipients, err := ReadSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("ReadSOPSConfig failed: %v", err)
	}

	expectedRecipients := []string{
		initialRecipients[0],
		newRecipient,
	}

	if len(recipients) != len(expectedRecipients) {
		t.Fatalf("expected %d recipients, got %d", len(expectedRecipients), len(recipients))
	}

	for i, expected := range expectedRecipients {
		if recipients[i] != expected {
			t.Errorf("recipient[%d]: expected %s, got %s", i, expected, recipients[i])
		}
	}
}

func TestAddRecipient_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()

	identity, _ := age.GenerateX25519Identity()
	recipient := identity.Recipient().String()

	err := CreateSOPSConfig(tmpDir, []string{recipient})
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	err = AddRecipient(tmpDir, recipient)
	if err == nil {
		t.Error("expected error when adding duplicate recipient")
	}

	if !strings.Contains(err.Error(), "recipient already exists") {
		t.Errorf("expected 'recipient already exists' error, got: %v", err)
	}
}

func TestRemoveRecipient(t *testing.T) {
	tmpDir := t.TempDir()

	identity1, _ := age.GenerateX25519Identity()
	identity2, _ := age.GenerateX25519Identity()
	recipients := []string{
		identity1.Recipient().String(),
		identity2.Recipient().String(),
	}

	err := CreateSOPSConfig(tmpDir, recipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	recipientToRemove := recipients[0]

	err = RemoveRecipient(tmpDir, recipientToRemove)
	if err != nil {
		t.Fatalf("RemoveRecipient failed: %v", err)
	}

	remainingRecipients, err := ReadSOPSConfig(tmpDir)
	if err != nil {
		t.Fatalf("ReadSOPSConfig failed: %v", err)
	}

	if len(remainingRecipients) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(remainingRecipients))
	}

	if remainingRecipients[0] != recipients[1] {
		t.Errorf("expected recipient %s, got %s", recipients[1], remainingRecipients[0])
	}
}

func TestRemoveRecipient_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	identity, _ := age.GenerateX25519Identity()
	recipients := []string{identity.Recipient().String()}

	err := CreateSOPSConfig(tmpDir, recipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	err = RemoveRecipient(tmpDir, "nonexistent")
	if err == nil {
		t.Error("expected error when removing non-existent recipient")
	}

	if !strings.Contains(err.Error(), "recipient not found") {
		t.Errorf("expected 'recipient not found' error, got: %v", err)
	}
}

func TestRemoveRecipient_LastRecipient(t *testing.T) {
	tmpDir := t.TempDir()

	identity, _ := age.GenerateX25519Identity()
	recipients := []string{identity.Recipient().String()}

	err := CreateSOPSConfig(tmpDir, recipients)
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	err = RemoveRecipient(tmpDir, recipients[0])
	if err == nil {
		t.Error("expected error when removing last recipient")
	}

	if !strings.Contains(err.Error(), "cannot remove last recipient") {
		t.Errorf("expected 'cannot remove last recipient' error, got: %v", err)
	}
}

func TestEncryptDecryptYAML(t *testing.T) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}

	publicKey := identity.Recipient().String()

	keyContent := identity.String() + "\n"
	keyPath := filepath.Join(tmpDir, "key.txt")
	err = os.WriteFile(keyPath, []byte(keyContent), 0600)
	if err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", keyPath); err != nil {
		t.Fatalf("failed to set SOPS_AGE_KEY_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	}()

	err = CreateSOPSConfig(tmpDir, []string{publicKey})
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	enc, err := NewEncryptor(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	type TestData struct {
		Message string `yaml:"message"`
		Count   int    `yaml:"count"`
	}

	originalData := TestData{
		Message: "secret message",
		Count:   42,
	}

	testFile := filepath.Join(tmpDir, "entries", "test.yaml")
	if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	err = enc.EncryptYAMLInMemory(originalData, testFile)
	if err != nil {
		t.Fatalf("EncryptYAMLInMemory failed: %v", err)
	}

	encryptedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read encrypted file: %v", err)
	}

	if !strings.Contains(string(encryptedContent), "sops") {
		t.Error("encrypted file does not contain SOPS metadata")
	}

	if strings.Contains(string(encryptedContent), "secret message") {
		t.Error("encrypted file contains plaintext data")
	}

	var decryptedData TestData
	err = enc.DecryptYAML(testFile, &decryptedData)
	if err != nil {
		t.Fatalf("DecryptYAML failed: %v", err)
	}

	if decryptedData.Message != originalData.Message {
		t.Errorf("expected message %s, got %s", originalData.Message, decryptedData.Message)
	}

	if decryptedData.Count != originalData.Count {
		t.Errorf("expected count %d, got %d", originalData.Count, decryptedData.Count)
	}
}

// TestEncryptFile tests encrypting an existing file
func TestEncryptFile(t *testing.T) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}

	publicKey := identity.Recipient().String()

	keyContent := identity.String() + "\n"
	keyPath := filepath.Join(tmpDir, "key.txt")
	err = os.WriteFile(keyPath, []byte(keyContent), 0600)
	if err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", keyPath); err != nil {
		t.Fatalf("failed to set SOPS_AGE_KEY_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	}()

	err = CreateSOPSConfig(tmpDir, []string{publicKey})
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	enc, err := NewEncryptor(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	testFile := filepath.Join(tmpDir, "entries", "test.yaml")
	if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	plaintext := "message: secret data\ncount: 123\n"
	err = os.WriteFile(testFile, []byte(plaintext), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err = enc.EncryptFile(testFile)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	encryptedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read encrypted file: %v", err)
	}

	if !strings.Contains(string(encryptedContent), "sops") {
		t.Error("encrypted file does not contain SOPS metadata")
	}

	if strings.Contains(string(encryptedContent), "secret data") {
		t.Error("encrypted file contains plaintext data")
	}
}

// TestDecryptFile tests decrypting a file
func TestDecryptFile(t *testing.T) {
	tmpDir := t.TempDir()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate age identity: %v", err)
	}

	publicKey := identity.Recipient().String()

	keyContent := identity.String() + "\n"
	keyPath := filepath.Join(tmpDir, "key.txt")
	err = os.WriteFile(keyPath, []byte(keyContent), 0600)
	if err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", keyPath); err != nil {
		t.Fatalf("failed to set SOPS_AGE_KEY_FILE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("SOPS_AGE_KEY_FILE"); err != nil {
			t.Errorf("failed to unset SOPS_AGE_KEY_FILE: %v", err)
		}
	}()

	err = CreateSOPSConfig(tmpDir, []string{publicKey})
	if err != nil {
		t.Fatalf("CreateSOPSConfig failed: %v", err)
	}

	enc, err := NewEncryptor(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	testFile := filepath.Join(tmpDir, "entries", "test.yaml")
	if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	expectedContent := "message: secret data\ncount: 123\n"
	err = os.WriteFile(testFile, []byte(expectedContent), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err = enc.EncryptFile(testFile)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	decryptedContent, err := enc.DecryptFile(testFile)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	if string(decryptedContent) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(decryptedContent))
	}
}

func generateRecipients(n int) []string {
	var recipients []string
	for range n {
		identity, _ := age.GenerateX25519Identity()
		recipients = append(recipients, identity.Recipient().String())
	}
	return recipients
}
