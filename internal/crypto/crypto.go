package crypto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/getsops/sops/v3/keyservice"
	sopsyaml "github.com/getsops/sops/v3/stores/yaml"
	"gopkg.in/yaml.v3"
)

// Encryptor handles encryption and decryption using SOPS
type Encryptor struct {
	journalPath string   // Path to journal directory (contains .sops.yaml)
	recipients  []string // Age public keys for encryption
}

// NewEncryptor creates a SOPS-based encryptor
// journalPath: path to journal directory (should contain .sops.yaml)
func NewEncryptor(journalPath string) (*Encryptor, error) {
	recipients, err := ReadSOPSConfig(journalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SOPS config: %w", err)
	}

	return &Encryptor{
		journalPath: journalPath,
		recipients:  recipients,
	}, nil
}

// EncryptFile encrypts a YAML file using SOPS
// filePath: absolute path to the file to encrypt
func (e *Encryptor) EncryptFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	store := sopsyaml.Store{}

	branches, err := store.LoadPlainFile(data)
	if err != nil {
		return fmt.Errorf("failed to load plain file: %w", err)
	}

	keyGroups, err := e.createKeyGroups()
	if err != nil {
		return fmt.Errorf("failed to create key groups: %w", err)
	}

	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups: keyGroups,
			Version:   "3.9.2",
		},
	}

	dataKey, errs := tree.GenerateDataKeyWithKeyServices(
		[]keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	)
	if len(errs) > 0 {
		return fmt.Errorf("failed to generate data key: %v", errs)
	}

	cipher := aes.NewCipher()
	mac, err := tree.Encrypt(dataKey, cipher)
	if err != nil {
		return fmt.Errorf("failed to encrypt tree: %w", err)
	}

	tree.Metadata.MessageAuthenticationCode, err = cipher.Encrypt(mac, dataKey, tree.Metadata.LastModified.Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return fmt.Errorf("failed to encrypt MAC: %w", err)
	}

	encryptedData, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return fmt.Errorf("failed to emit encrypted YAML: %w", err)
	}

	if err := os.WriteFile(filePath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// DecryptFile decrypts a SOPS-encrypted file and returns the content
// filePath: absolute path to the encrypted file
func (e *Encryptor) DecryptFile(filePath string) ([]byte, error) {
	cleartext, err := decrypt.File(filePath, "yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt file: %w", err)
	}

	return cleartext, nil
}

// EncryptYAMLInMemory encrypts YAML data in memory and writes only the encrypted result
// data: the data structure to encrypt
// filePath: where to write the encrypted file
func (e *Encryptor) EncryptYAMLInMemory(data any, filePath string) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	store := sopsyaml.Store{}

	branches, err := store.LoadPlainFile(yamlData)
	if err != nil {
		return fmt.Errorf("failed to load plain YAML: %w", err)
	}

	keyGroups, err := e.createKeyGroups()
	if err != nil {
		return fmt.Errorf("failed to create key groups: %w", err)
	}

	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups: keyGroups,
			Version:   "3.9.2",
		},
	}

	dataKey, errs := tree.GenerateDataKeyWithKeyServices(
		[]keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	)
	if len(errs) > 0 {
		return fmt.Errorf("failed to generate data key: %v", errs)
	}

	cipher := aes.NewCipher()
	mac, err := tree.Encrypt(dataKey, cipher)
	if err != nil {
		return fmt.Errorf("failed to encrypt tree: %w", err)
	}

	tree.Metadata.MessageAuthenticationCode, err = cipher.Encrypt(mac, dataKey, tree.Metadata.LastModified.Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return fmt.Errorf("failed to encrypt MAC: %w", err)
	}

	encryptedData, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return fmt.Errorf("failed to emit encrypted YAML: %w", err)
	}

	if err := os.WriteFile(filePath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// VerifyEncryptedFile verifies a file can be decrypted with current keys
// Returns nil if successful, error otherwise
func (e *Encryptor) VerifyEncryptedFile(filePath string) error {
	_, err := e.DecryptFile(filePath)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}
	return nil
}

// DecryptYAML decrypts a SOPS-encrypted YAML file and unmarshals it
// filePath: path to encrypted file
// target: pointer to struct to unmarshal into
func (e *Encryptor) DecryptYAML(filePath string, target any) error {
	decrypted, err := e.DecryptFile(filePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(decrypted, target); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return nil
}

// createKeyGroups creates SOPS key groups from age recipients
func (e *Encryptor) createKeyGroups() ([]sops.KeyGroup, error) {
	var keyGroup sops.KeyGroup

	for _, recipient := range e.recipients {
		ageRecipient, err := age.ParseX25519Recipient(recipient)
		if err != nil {
			return nil, fmt.Errorf("invalid age recipient %s: %w", recipient, err)
		}

		keyGroup = append(keyGroup, &sopsage.MasterKey{
			Recipient: ageRecipient.String(),
		})
	}

	if len(keyGroup) == 0 {
		return nil, fmt.Errorf("no valid recipients found")
	}

	return []sops.KeyGroup{keyGroup}, nil
}

// SOPSConfig represents the .sops.yaml configuration file
type SOPSConfig struct {
	CreationRules []CreationRule `yaml:"creation_rules"`
}

// CreationRule represents a single rule in .sops.yaml
type CreationRule struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
}

// ValidateRecipient validates that a recipient is a valid age public key
func ValidateRecipient(recipient string) error {
	_, err := age.ParseX25519Recipient(recipient)
	if err != nil {
		return fmt.Errorf("invalid age public key: %w", err)
	}
	return nil
}

// CreateSOPSConfig creates or updates a .sops.yaml file with age recipients
// journalPath: path to journal directory
// recipients: list of age public keys
func CreateSOPSConfig(journalPath string, recipients []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	for _, recipient := range recipients {
		if err := ValidateRecipient(recipient); err != nil {
			return fmt.Errorf("recipient %s: %w", recipient, err)
		}
	}

	config := SOPSConfig{
		CreationRules: []CreationRule{
			{
				PathRegex: "index\\.yaml$",
				Age:       strings.Join(recipients, ","),
			},
			{
				PathRegex: "entries/.*\\.yaml$",
				Age:       strings.Join(recipients, ","),
			},
		},
	}

	configPath := filepath.Join(journalPath, ".sops.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal SOPS config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write .sops.yaml: %w", err)
	}

	return nil
}

// ReadSOPSConfig reads the .sops.yaml file and returns the recipients
func ReadSOPSConfig(journalPath string) ([]string, error) {
	configPath := filepath.Join(journalPath, ".sops.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .sops.yaml: %w", err)
	}

	var config SOPSConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse .sops.yaml: %w", err)
	}

	if len(config.CreationRules) == 0 {
		return nil, fmt.Errorf("no creation rules found in .sops.yaml")
	}

	ageRecipients := config.CreationRules[0].Age
	if ageRecipients == "" {
		return nil, fmt.Errorf("no age recipients found in .sops.yaml")
	}

	// Split comma-separated recipients and trim whitespace
	recipients := strings.Split(ageRecipients, ",")
	for i, r := range recipients {
		recipients[i] = strings.TrimSpace(r)
	}

	return recipients, nil
}

// AddRecipient adds a new age public key to the .sops.yaml file
func AddRecipient(journalPath string, newRecipient string) error {
	recipients, err := ReadSOPSConfig(journalPath)
	if err != nil {
		return err
	}

	for _, r := range recipients {
		if r == newRecipient {
			return fmt.Errorf("recipient already exists")
		}
	}

	recipients = append(recipients, newRecipient)
	return CreateSOPSConfig(journalPath, recipients)
}

// RemoveRecipient removes an age public key from the .sops.yaml file
func RemoveRecipient(journalPath string, recipientToRemove string) error {
	recipients, err := ReadSOPSConfig(journalPath)
	if err != nil {
		return err
	}

	found := false
	newRecipients := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if r != recipientToRemove {
			newRecipients = append(newRecipients, r)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("recipient not found")
	}

	if len(newRecipients) == 0 {
		return fmt.Errorf("cannot remove last recipient")
	}

	return CreateSOPSConfig(journalPath, newRecipients)
}

// BackupSOPSConfig creates a timestamped backup of .sops.yaml
// Returns the backup file path for later restoration
func BackupSOPSConfig(journalPath string) (string, error) {
	configPath := filepath.Join(journalPath, ".sops.yaml")

	if _, err := os.Stat(configPath); err != nil {
		return "", fmt.Errorf("failed to stat .sops.yaml: %w", err)
	}

	timestamp := fmt.Sprintf("%d", os.Getpid())
	backupPath := filepath.Join(journalPath, fmt.Sprintf(".sops.yaml.backup.%s", timestamp))

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .sops.yaml: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// RestoreSOPSConfig restores .sops.yaml from backup and removes the backup file
func RestoreSOPSConfig(journalPath string, backupPath string) error {
	configPath := filepath.Join(journalPath, ".sops.yaml")

	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to restore .sops.yaml: %w", err)
	}

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to remove backup: %w", err)
	}

	return nil
}

// RemoveBackup deletes the backup file after successful operation
func RemoveBackup(backupPath string) error {
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to remove backup: %w", err)
	}
	return nil
}

// PrepareAddRecipient validates and returns new recipient list for adding a recipient
// Does not modify .sops.yaml - that happens in the transaction
func PrepareAddRecipient(journalPath string, newRecipient string) ([]string, error) {
	recipients, err := ReadSOPSConfig(journalPath)
	if err != nil {
		return nil, err
	}

	if err := ValidateRecipient(newRecipient); err != nil {
		return nil, err
	}

	for _, r := range recipients {
		if r == newRecipient {
			return nil, fmt.Errorf("recipient already exists")
		}
	}

	return append(recipients, newRecipient), nil
}

// PrepareRemoveRecipient validates and returns new recipient list for removing a recipient
// Does not modify .sops.yaml - that happens in the transaction
func PrepareRemoveRecipient(journalPath string, recipientToRemove string) ([]string, error) {
	recipients, err := ReadSOPSConfig(journalPath)
	if err != nil {
		return nil, err
	}

	found := false
	newRecipients := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if r != recipientToRemove {
			newRecipients = append(newRecipients, r)
		} else {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("recipient not found")
	}

	if len(newRecipients) == 0 {
		return nil, fmt.Errorf("cannot remove last recipient")
	}

	return newRecipients, nil
}
