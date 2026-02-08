package services

import (
	"os"
	"path/filepath"
	"testing"

	"micropanel/internal/config"
)

func TestFileService_ValidatePath_PathTraversal(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	sitesDir := filepath.Join(tempDir, "sites")
	os.MkdirAll(filepath.Join(sitesDir, "1", "public"), 0755)
	os.MkdirAll(filepath.Join(sitesDir, "1", "public_evil"), 0755)
	os.MkdirAll(filepath.Join(sitesDir, "2", "public"), 0755)

	// Create a secret file outside public
	secretPath := filepath.Join(sitesDir, "1", "secret.txt")
	os.WriteFile(secretPath, []byte("secret"), 0644)

	cfg := &config.Config{}
	cfg.Sites.Path = sitesDir

	fs := NewFileService(cfg)

	tests := []struct {
		name      string
		siteID    int64
		path      string
		wantErr   bool
		errType   error
	}{
		// Valid paths
		{"root path", 1, "/", false, nil},
		{"empty path", 1, "", false, nil},
		{"simple file", 1, "/index.html", false, nil},
		{"nested path", 1, "/css/style.css", false, nil},
		{"dot in filename", 1, "/file.min.js", false, nil},

		// Path traversal attempts (without leading slash - the actual attack vector)
		{"traversal no slash", 1, "../secret.txt", true, ErrFilePathTraversal},
		{"double traversal no slash", 1, "../../etc/passwd", true, ErrFilePathTraversal},
		{"nested traversal no slash", 1, "subdir/../../secret.txt", true, ErrFilePathTraversal},
		{"deep traversal", 1, "a/b/c/../../../../secret.txt", true, ErrFilePathTraversal},

		// Note: Paths starting with / are cleaned by filepath.Clean which resolves
		// /../ against root, making them safe. These become /something -> something
		// which stays inside sandbox. This is expected Go behavior.
		{"slash anchored cleaned", 1, "/../secret.txt", false, nil}, // becomes /secret.txt -> secret.txt (inside sandbox)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fs.ValidatePath(tt.siteID, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
			if tt.wantErr && tt.errType != nil && err != tt.errType {
				t.Errorf("ValidatePath(%q) error = %v, want %v", tt.path, err, tt.errType)
			}
		})
	}
}

func TestFileService_ValidatePath_NoEscapeToParent(t *testing.T) {
	tempDir := t.TempDir()
	sitesDir := filepath.Join(tempDir, "sites")

	// Create site 1 public directory
	site1Public := filepath.Join(sitesDir, "1", "public")
	os.MkdirAll(site1Public, 0755)

	// Create a sensitive file at sites/1 level (outside public)
	sensitiveFile := filepath.Join(sitesDir, "1", "config.json")
	os.WriteFile(sensitiveFile, []byte(`{"db_password": "secret"}`), 0644)

	// Create sites/1/public_backup (similar name to public)
	backupDir := filepath.Join(sitesDir, "1", "public_backup")
	os.MkdirAll(backupDir, 0755)
	os.WriteFile(filepath.Join(backupDir, "backup.sql"), []byte("DROP TABLE users;"), 0644)

	cfg := &config.Config{}
	cfg.Sites.Path = sitesDir

	fs := NewFileService(cfg)

	// Traversal attempts WITHOUT leading slash (actual attack vector)
	// Paths with leading / get cleaned by filepath.Clean resolving against root
	escapeAttempts := []string{
		"../config.json",
		"../public_backup/backup.sql",
		"subdir/../../../config.json",
		"../../../etc/passwd",
	}

	for _, path := range escapeAttempts {
		t.Run(path, func(t *testing.T) {
			_, err := fs.ValidatePath(1, path)
			if err != ErrFilePathTraversal {
				t.Errorf("Expected ErrFilePathTraversal for path %q, got %v", path, err)
			}
		})
	}
}

func TestFileService_Read_CannotReadOutsidePublic(t *testing.T) {
	tempDir := t.TempDir()
	sitesDir := filepath.Join(tempDir, "sites")

	// Create site structure
	site1Public := filepath.Join(sitesDir, "1", "public")
	os.MkdirAll(site1Public, 0755)

	// Create files
	os.WriteFile(filepath.Join(site1Public, "index.html"), []byte("<html>OK</html>"), 0644)
	os.WriteFile(filepath.Join(sitesDir, "1", "secret.txt"), []byte("SECRET"), 0644)

	cfg := &config.Config{}
	cfg.Sites.Path = sitesDir

	fs := NewFileService(cfg)

	// Should be able to read public file
	content, err := fs.Read(1, "/index.html")
	if err != nil {
		t.Errorf("Should read public file: %v", err)
	}
	if string(content) != "<html>OK</html>" {
		t.Errorf("Wrong content: %s", content)
	}

	// Should NOT be able to read secret file (without leading slash - actual attack vector)
	_, err = fs.Read(1, "../secret.txt")
	if err != ErrFilePathTraversal {
		t.Errorf("Should block reading secret file with '../secret.txt', got: %v", err)
	}

	// Also test deeper traversal
	_, err = fs.Read(1, "subdir/../../secret.txt")
	if err != ErrFilePathTraversal {
		t.Errorf("Should block reading with nested traversal, got: %v", err)
	}
}
