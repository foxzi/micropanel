package services

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"micropanel/internal/config"
)

var (
	ErrFilePathTraversal = errors.New("path traversal detected")
	ErrFileNotFound      = errors.New("file not found")
	ErrIsDirectory       = errors.New("path is a directory")
	ErrIsNotDirectory    = errors.New("path is not a directory")
	ErrFileTooBig        = errors.New("file too large")
	ErrFileInvalidPath   = errors.New("invalid path")
	ErrFileExists        = errors.New("file already exists")
	ErrCannotDelete      = errors.New("cannot delete this path")
)

const (
	MaxEditFileSize = 5 * 1024 * 1024  // 5MB
	MaxUploadSize   = 10 * 1024 * 1024 // 10MB
)

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

type FileService struct {
	config *config.Config
}

func NewFileService(cfg *config.Config) *FileService {
	return &FileService{
		config: cfg,
	}
}

// GetSitePath returns the base path for a site
func (s *FileService) GetSitePath(siteID int64) string {
	return filepath.Join(s.config.Sites.Path, fmt.Sprintf("%d", siteID), "public")
}

// ValidatePath checks if the path is within the site directory (sandbox check)
func (s *FileService) ValidatePath(siteID int64, relativePath string) (string, error) {
	basePath := s.GetSitePath(siteID)

	// Clean and normalize the path
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." {
		cleanPath = ""
	}

	// Remove leading slash
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	// Build full path
	fullPath := filepath.Join(basePath, cleanPath)

	// Resolve to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", ErrFileInvalidPath
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", ErrFileInvalidPath
	}

	// Check if path is within base directory
	if !strings.HasPrefix(absPath, absBase) {
		return "", ErrFilePathTraversal
	}

	return absPath, nil
}

// List returns files and directories at the given path
func (s *FileService) List(siteID int64, relativePath string) ([]FileInfo, error) {
	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return nil, err
	}

	// Check if path exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		// If public directory doesn't exist, create it
		if relativePath == "" || relativePath == "/" {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return nil, fmt.Errorf("create public dir: %w", err)
			}
			return []FileInfo{}, nil
		}
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, ErrIsNotDirectory
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	basePath := s.GetSitePath(siteID)

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := filepath.Join(fullPath, entry.Name())
		relPath, _ := filepath.Rel(basePath, entryPath)

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    "/" + relPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return files, nil
}

// Read returns the content of a file
func (s *FileService) Read(siteID int64, relativePath string) ([]byte, error) {
	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, ErrIsDirectory
	}

	if info.Size() > MaxEditFileSize {
		return nil, ErrFileTooBig
	}

	return os.ReadFile(fullPath)
}

// Write saves content to a file
func (s *FileService) Write(siteID int64, relativePath string, content []byte) error {
	if len(content) > MaxEditFileSize {
		return ErrFileTooBig
	}

	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(fullPath, content, 0644)
}

// CreateFile creates a new empty file
func (s *FileService) CreateFile(siteID int64, relativePath string) error {
	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return err
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return ErrFileExists
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Create empty file
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	return f.Close()
}

// CreateDirectory creates a new directory
func (s *FileService) CreateDirectory(siteID int64, relativePath string) error {
	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return err
	}

	// Check if already exists
	if _, err := os.Stat(fullPath); err == nil {
		return ErrFileExists
	}

	return os.MkdirAll(fullPath, 0755)
}

// Delete removes a file or directory
func (s *FileService) Delete(siteID int64, relativePath string) error {
	// Prevent deleting root
	if relativePath == "" || relativePath == "/" {
		return ErrCannotDelete
	}

	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return err
	}

	// Check if exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return ErrFileNotFound
	}

	return os.RemoveAll(fullPath)
}

// Rename moves/renames a file or directory
func (s *FileService) Rename(siteID int64, oldPath, newPath string) error {
	oldFullPath, err := s.ValidatePath(siteID, oldPath)
	if err != nil {
		return err
	}

	newFullPath, err := s.ValidatePath(siteID, newPath)
	if err != nil {
		return err
	}

	// Check if source exists
	if _, err := os.Stat(oldFullPath); os.IsNotExist(err) {
		return ErrFileNotFound
	}

	// Check if destination already exists
	if _, err := os.Stat(newFullPath); err == nil {
		return ErrFileExists
	}

	// Ensure parent directory of destination exists
	dir := filepath.Dir(newFullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.Rename(oldFullPath, newFullPath)
}

// Upload saves an uploaded file
func (s *FileService) Upload(siteID int64, relativePath string, reader io.Reader, size int64) error {
	if size > MaxUploadSize {
		return ErrFileTooBig
	}

	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Create file
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Copy with size limit
	_, err = io.CopyN(f, reader, MaxUploadSize)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// GetFileInfo returns information about a file or directory
func (s *FileService) GetFileInfo(siteID int64, relativePath string) (*FileInfo, error) {
	fullPath, err := s.ValidatePath(siteID, relativePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}

	basePath := s.GetSitePath(siteID)
	relPath, _ := filepath.Rel(basePath, fullPath)

	return &FileInfo{
		Name:    info.Name(),
		Path:    "/" + relPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
	}, nil
}

// IsTextFile checks if a file is likely a text file based on extension
func (s *FileService) IsTextFile(filename string) bool {
	textExtensions := map[string]bool{
		".html": true, ".htm": true, ".css": true, ".js": true,
		".json": true, ".xml": true, ".svg": true, ".txt": true,
		".md": true, ".yml": true, ".yaml": true, ".toml": true,
		".ini": true, ".conf": true, ".cfg": true, ".sh": true,
		".php": true, ".py": true, ".rb": true, ".go": true,
		".ts": true, ".tsx": true, ".jsx": true, ".vue": true,
		".scss": true, ".sass": true, ".less": true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	return textExtensions[ext]
}

// IsImageFile checks if a file is an image based on extension
func (s *FileService) IsImageFile(filename string) bool {
	imageExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".ico": true, ".bmp": true, ".svg": true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	return imageExtensions[ext]
}
