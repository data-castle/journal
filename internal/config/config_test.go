package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_EmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "journal-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	}()

	// Create empty config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("failed to write empty config: %v", err)
	}

	// Override config path for this test
	origFunc := GetConfigPathFunc
	GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPathFunc = origFunc }()

	// Should return error for empty file
	_, err = LoadConfig()
	if err == nil {
		t.Error("LoadConfig() should return error for empty file")
	}
	if err != nil && err.Error() != "config file is empty (possibly corrupted)" {
		t.Errorf("LoadConfig() wrong error message: %v", err)
	}
}

func TestLoadConfig_CorruptedJournalsField(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "journal-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	}()

	// Create config file with null journals field
	configPath := filepath.Join(tmpDir, "config.yaml")
	corruptedYAML := `default_journal: personal
journals: null
`
	if err := os.WriteFile(configPath, []byte(corruptedYAML), 0600); err != nil {
		t.Fatalf("failed to write corrupted config: %v", err)
	}

	// Override config path for this test
	origFunc := GetConfigPathFunc
	GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPathFunc = origFunc }()

	// Should return error for corrupted journals field
	_, err = LoadConfig()
	if err == nil {
		t.Error("LoadConfig() should return error for corrupted journals field")
	}
	if err != nil && err.Error() != "config file is corrupted: 'journals' field is null" {
		t.Errorf("LoadConfig() wrong error message: %v", err)
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "journal-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	}()

	// Use non-existent config path
	configPath := filepath.Join(tmpDir, "nonexistent", "config.yaml")

	// Override config path for this test
	origFunc := GetConfigPathFunc
	GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPathFunc = origFunc }()

	// Should return empty config (no error)
	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() should not error for non-existent file: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}
	if cfg.Journals == nil {
		t.Error("LoadConfig() should initialize Journals map")
	}
	if len(cfg.Journals) != 0 {
		t.Error("LoadConfig() should return empty Journals map")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "journal-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	}()

	// Create valid config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	validYAML := `default_journal: personal
journals:
  personal:
    name: personal
    path: /home/user/journal
  work:
    name: work
    path: /home/user/work-journal
`
	if err := os.WriteFile(configPath, []byte(validYAML), 0600); err != nil {
		t.Fatalf("failed to write valid config: %v", err)
	}

	// Override config path for this test
	origFunc := GetConfigPathFunc
	GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPathFunc = origFunc }()

	// Should load successfully
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if cfg.DefaultJournal != "personal" {
		t.Errorf("DefaultJournal = %v, want personal", cfg.DefaultJournal)
	}
	if len(cfg.Journals) != 2 {
		t.Errorf("len(Journals) = %v, want 2", len(cfg.Journals))
	}

	personal, exists := cfg.Journals["personal"]
	if !exists {
		t.Fatal("personal journal not found")
	}
	if personal.Name != "personal" {
		t.Errorf("personal.Name = %v, want personal", personal.Name)
	}
	if personal.Path != "/home/user/journal" {
		t.Errorf("personal.Path = %v, want /home/user/journal", personal.Path)
	}
}

func TestConfig_Save(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "journal-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	}()

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Override config path for this test
	origFunc := GetConfigPathFunc
	GetConfigPathFunc = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPathFunc = origFunc }()

	cfg := &Config{
		DefaultJournal: "test",
		Journals: map[string]*Journal{
			"test": {
				Name: "test",
				Path: "/test/path",
			},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() after Save() failed: %v", err)
	}

	if loaded.DefaultJournal != cfg.DefaultJournal {
		t.Errorf("DefaultJournal = %v, want %v", loaded.DefaultJournal, cfg.DefaultJournal)
	}
	if len(loaded.Journals) != len(cfg.Journals) {
		t.Errorf("len(Journals) = %v, want %v", len(loaded.Journals), len(cfg.Journals))
	}
}

func TestConfig_AddJournal(t *testing.T) {
	tests := []struct {
		name        string
		initial     *Config
		journal     *Journal
		wantErr     bool
		errContains string
	}{
		{
			name: "add first journal",
			initial: &Config{
				Journals: make(map[string]*Journal),
			},
			journal: &Journal{
				Name: "personal",
				Path: "/home/user/journal",
			},
			wantErr: false,
		},
		{
			name: "add second journal",
			initial: &Config{
				DefaultJournal: "first",
				Journals: map[string]*Journal{
					"first": {Name: "first", Path: "/first"},
				},
			},
			journal: &Journal{
				Name: "second",
				Path: "/second",
			},
			wantErr: false,
		},
		{
			name: "add duplicate journal",
			initial: &Config{
				Journals: map[string]*Journal{
					"existing": {Name: "existing", Path: "/existing"},
				},
			},
			journal: &Journal{
				Name: "existing",
				Path: "/new/path",
			},
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name: "add journal with empty name",
			initial: &Config{
				Journals: make(map[string]*Journal),
			},
			journal: &Journal{
				Name: "",
				Path: "/path",
			},
			wantErr:     true,
			errContains: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.AddJournal(tt.journal)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddJournal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("AddJournal() error = %v, should contain %v", err, tt.errContains)
				}
				return
			}

			if !tt.wantErr {
				if _, exists := tt.initial.Journals[tt.journal.Name]; !exists {
					t.Error("journal was not added to map")
				}
				if len(tt.initial.Journals) == 1 && tt.initial.DefaultJournal != tt.journal.Name {
					t.Error("first journal should become default")
				}
			}
		})
	}
}

func TestConfig_GetJournal(t *testing.T) {
	cfg := &Config{
		Journals: map[string]*Journal{
			"personal": {Name: "personal", Path: "/personal"},
			"work":     {Name: "work", Path: "/work"},
		},
	}

	tests := []struct {
		name    string
		journal string
		wantErr bool
	}{
		{
			name:    "get existing journal",
			journal: "personal",
			wantErr: false,
		},
		{
			name:    "get non-existent journal",
			journal: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.GetJournal(tt.journal)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetJournal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Error("GetJournal() returned nil")
					return
				}
				if got.Name != tt.journal {
					t.Errorf("GetJournal() returned wrong journal: %v", got.Name)
				}
			}
		})
	}
}

func TestConfig_GetJournal_NoJournals(t *testing.T) {
	cfg := &Config{
		Journals: make(map[string]*Journal),
	}

	_, err := cfg.GetJournal("any")
	if err == nil {
		t.Error("GetJournal() should error when no journals configured")
	}
	if !contains(err.Error(), "no journals configured") {
		t.Errorf("GetJournal() wrong error: %v", err)
	}
}

func TestConfig_GetDefaultJournal(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		wantMsg string
	}{
		{
			name: "get valid default",
			cfg: &Config{
				DefaultJournal: "personal",
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
				},
			},
			wantErr: false,
		},
		{
			name: "no default set",
			cfg: &Config{
				DefaultJournal: "",
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
				},
			},
			wantErr: true,
			wantMsg: "no default journal set",
		},
		{
			name: "no journals configured",
			cfg: &Config{
				Journals: make(map[string]*Journal),
			},
			wantErr: true,
			wantMsg: "no journals configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.GetDefaultJournal()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultJournal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.wantMsg != "" {
				if !contains(err.Error(), tt.wantMsg) {
					t.Errorf("GetDefaultJournal() error = %v, should contain %v", err, tt.wantMsg)
				}
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Error("GetDefaultJournal() returned nil")
					return
				}
				if got.Name != tt.cfg.DefaultJournal {
					t.Errorf("GetDefaultJournal() returned wrong journal: %v", got.Name)
				}
			}
		})
	}
}

func TestConfig_SetDefaultJournal(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		journal string
		wantErr bool
	}{
		{
			name: "set valid default",
			cfg: &Config{
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
					"work":     {Name: "work", Path: "/work"},
				},
			},
			journal: "work",
			wantErr: false,
		},
		{
			name: "set non-existent journal",
			cfg: &Config{
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
				},
			},
			journal: "nonexistent",
			wantErr: true,
		},
		{
			name: "no journals configured",
			cfg: &Config{
				Journals: make(map[string]*Journal),
			},
			journal: "any",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.SetDefaultJournal(tt.journal)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetDefaultJournal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.cfg.DefaultJournal != tt.journal {
					t.Errorf("DefaultJournal = %v, want %v", tt.cfg.DefaultJournal, tt.journal)
				}
			}
		})
	}
}

func TestConfig_RemoveJournal(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		journal string
		wantErr bool
	}{
		{
			name: "remove non-default journal",
			cfg: &Config{
				DefaultJournal: "personal",
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
					"work":     {Name: "work", Path: "/work"},
				},
			},
			journal: "work",
			wantErr: false,
		},
		{
			name: "remove default journal",
			cfg: &Config{
				DefaultJournal: "personal",
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
					"work":     {Name: "work", Path: "/work"},
				},
			},
			journal: "personal",
			wantErr: true,
		},
		{
			name: "remove non-existent journal",
			cfg: &Config{
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
				},
			},
			journal: "nonexistent",
			wantErr: true,
		},
		{
			name: "no journals configured",
			cfg: &Config{
				Journals: make(map[string]*Journal),
			},
			journal: "any",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDefault := tt.cfg.DefaultJournal
			err := tt.cfg.RemoveJournal(tt.journal)

			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveJournal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if _, exists := tt.cfg.Journals[tt.journal]; exists {
					t.Error("journal was not removed")
				}
				if oldDefault == tt.journal && tt.cfg.DefaultJournal != "" {
					t.Error("default journal should be cleared")
				}
			}
		})
	}
}

func TestConfig_ListJournals(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want int
	}{
		{
			name: "list multiple journals",
			cfg: &Config{
				Journals: map[string]*Journal{
					"personal": {Name: "personal", Path: "/personal"},
					"work":     {Name: "work", Path: "/work"},
					"travel":   {Name: "travel", Path: "/travel"},
				},
			},
			want: 3,
		},
		{
			name: "list empty journals",
			cfg: &Config{
				Journals: make(map[string]*Journal),
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ListJournals()

			if len(got) != tt.want {
				t.Errorf("ListJournals() returned %d journals, want %d", len(got), tt.want)
			}

			// Verify all journal names are present
			for name := range tt.cfg.Journals {
				found := false
				for _, journalName := range got {
					if journalName == name {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("journal %s not in ListJournals() result", name)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
