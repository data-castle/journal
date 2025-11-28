package crypto

import (
	"fmt"
	"strings"
)

// ReEncryptResult tracks the outcome of re-encryption
type ReEncryptResult struct {
	TotalFiles      int
	SuccessfulFiles int
	FailedFiles     []FileError
	IndexSuccess    bool
	IndexError      error
}

// FileError tracks individual file encryption failures
type FileError struct {
	FilePath string
	Error    error
}

// FormatErrors returns a human-readable summary of failures
func (r *ReEncryptResult) FormatErrors() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Total files: %d\n", r.TotalFiles)
	fmt.Fprintf(&sb, "Successful: %d\n", r.SuccessfulFiles)
	fmt.Fprintf(&sb, "Failed: %d\n", len(r.FailedFiles))

	if !r.IndexSuccess {
		fmt.Fprintf(&sb, "Index encryption: FAILED - %v\n", r.IndexError)
	} else {
		fmt.Fprintf(&sb, "Index encryption: SUCCESS\n")
	}

	if len(r.FailedFiles) > 0 {
		fmt.Fprintf(&sb, "\nFailed files:\n")
		for _, fe := range r.FailedFiles {
			fmt.Fprintf(&sb, "  - %s: %v\n", fe.FilePath, fe.Error)
		}
	}

	return sb.String()
}

// TransactionalReEncrypt performs atomic re-encryption with rollback
// This function ensures that either all files are successfully re-encrypted or
// the operation is rolled back completely
func TransactionalReEncrypt(
	journalPath string,
	newRecipients []string,
	listEntriesFunc func() ([]string, error),
	reEncryptEntryFunc func(string) error,
	reEncryptIndexFunc func() error,
) (*ReEncryptResult, error) {
	result := &ReEncryptResult{
		IndexSuccess: false,
	}

	// Step 1: Create backup of .sops.yaml
	backupPath, err := BackupSOPSConfig(journalPath)
	if err != nil {
		return result, fmt.Errorf("failed to backup .sops.yaml: %w", err)
	}

	// Step 2: Update .sops.yaml with new recipients
	if err := CreateSOPSConfig(journalPath, newRecipients); err != nil {
		if rerr := RestoreSOPSConfig(journalPath, backupPath); rerr != nil {
			return result, fmt.Errorf("failed to update .sops.yaml: %w (rollback also failed: %v)", err, rerr)
		}
		return result, fmt.Errorf("failed to update .sops.yaml: %w", err)
	}

	// Step 3: List all entry files
	files, err := listEntriesFunc()
	if err != nil {
		if rerr := RestoreSOPSConfig(journalPath, backupPath); rerr != nil {
			return result, fmt.Errorf("failed to list entries: %w (rollback also failed: %v)", err, rerr)
		}
		return result, fmt.Errorf("failed to list entries: %w", err)
	}

	result.TotalFiles = len(files)

	// Step 4: Re-encrypt all entries (continue through failures to collect all errors)
	for _, filePath := range files {
		if err := reEncryptEntryFunc(filePath); err != nil {
			result.FailedFiles = append(result.FailedFiles, FileError{
				FilePath: filePath,
				Error:    err,
			})
		} else {
			result.SuccessfulFiles++
		}
	}

	// Step 5: Re-encrypt index
	if err := reEncryptIndexFunc(); err != nil {
		result.IndexError = err
		result.IndexSuccess = false
	} else {
		result.IndexSuccess = true
	}

	// Step 6: Check if ALL operations succeeded
	if len(result.FailedFiles) > 0 || !result.IndexSuccess {
		if err := RestoreSOPSConfig(journalPath, backupPath); err != nil {
			return result, fmt.Errorf("re-encryption failed AND rollback failed: %w\nOriginal error: %s",
				err, result.FormatErrors())
		}

		return result, fmt.Errorf("re-encryption failed, rolled back .sops.yaml")
	}

	// Step 7: Success! Remove backup
	if err := RemoveBackup(backupPath); err != nil {
		fmt.Printf("Warning: failed to remove backup file %s: %v\n", backupPath, err)
	}

	return result, nil
}
