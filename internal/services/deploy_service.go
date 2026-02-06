package services

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"micropanel/internal/config"
	"micropanel/internal/models"
	"micropanel/internal/repository"
)

const (
	MaxZipSize     = 100 * 1024 * 1024 // 100MB
	MaxFileSize    = 10 * 1024 * 1024  // 10MB per file
	MaxFiles       = 10000
	MaxPathLength  = 500
)

var (
	ErrZipTooLarge     = errors.New("zip file too large")
	ErrTooManyFiles    = errors.New("too many files in archive")
	ErrPathTraversal   = errors.New("path traversal detected")
	ErrInvalidPath     = errors.New("invalid file path")
	ErrSymlinkDetected = errors.New("symlinks not allowed")
	ErrFileTooLarge    = errors.New("file too large")
)

type DeployService struct {
	config     *config.Config
	deployRepo *repository.DeployRepository
	siteRepo   *repository.SiteRepository
}

func NewDeployService(cfg *config.Config, deployRepo *repository.DeployRepository, siteRepo *repository.SiteRepository) *DeployService {
	return &DeployService{
		config:     cfg,
		deployRepo: deployRepo,
		siteRepo:   siteRepo,
	}
}

func (s *DeployService) Deploy(siteID, userID int64, filename string, zipReader io.Reader, size int64) (*models.Deploy, error) {
	// Check size
	if size > MaxZipSize {
		return nil, ErrZipTooLarge
	}

	// Create deploy record
	deploy := &models.Deploy{
		SiteID:   siteID,
		UserID:   userID,
		Filename: filename,
		Status:   models.DeployStatusPending,
	}
	if err := s.deployRepo.Create(deploy); err != nil {
		return nil, fmt.Errorf("create deploy record: %w", err)
	}

	// Process deploy
	if err := s.processDeploy(deploy, zipReader, size); err != nil {
		s.deployRepo.UpdateStatus(deploy.ID, models.DeployStatusFailed, err.Error())
		deploy.Status = models.DeployStatusFailed
		deploy.ErrorMessage = err.Error()
		return deploy, err
	}

	s.deployRepo.UpdateStatus(deploy.ID, models.DeployStatusSuccess, "")
	deploy.Status = models.DeployStatusSuccess
	return deploy, nil
}

func (s *DeployService) processDeploy(deploy *models.Deploy, zipReader io.Reader, size int64) error {
	sitePath := filepath.Join(s.config.Sites.Path, fmt.Sprintf("%d", deploy.SiteID))
	publicPath := filepath.Join(sitePath, "public")
	publicNewPath := filepath.Join(sitePath, "public_new")
	publicPrevPath := filepath.Join(sitePath, "public_prev")
	deploysPath := filepath.Join(sitePath, "deploys")

	// Ensure directories exist
	for _, dir := range []string{sitePath, deploysPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	// Save ZIP to deploys directory
	zipFilename := fmt.Sprintf("%d_%s", time.Now().Unix(), deploy.Filename)
	zipPath := filepath.Join(deploysPath, zipFilename)

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}

	written, err := io.Copy(zipFile, io.LimitReader(zipReader, MaxZipSize+1))
	zipFile.Close()
	if err != nil {
		os.Remove(zipPath)
		return fmt.Errorf("save zip file: %w", err)
	}
	if written > MaxZipSize {
		os.Remove(zipPath)
		return ErrZipTooLarge
	}

	// Clean up public_new if exists
	os.RemoveAll(publicNewPath)

	// Extract ZIP
	if err := s.extractZip(zipPath, publicNewPath); err != nil {
		os.RemoveAll(publicNewPath)
		return fmt.Errorf("extract zip: %w", err)
	}

	// Atomic swap
	// 1. Remove old public_prev
	os.RemoveAll(publicPrevPath)

	// 2. Move current public to public_prev (if exists)
	if _, err := os.Stat(publicPath); err == nil {
		if err := os.Rename(publicPath, publicPrevPath); err != nil {
			os.RemoveAll(publicNewPath)
			return fmt.Errorf("backup current public: %w", err)
		}
	}

	// 3. Move public_new to public
	if err := os.Rename(publicNewPath, publicPath); err != nil {
		// Try to restore
		if _, err2 := os.Stat(publicPrevPath); err2 == nil {
			os.Rename(publicPrevPath, publicPath)
		}
		return fmt.Errorf("activate new public: %w", err)
	}

	return nil
}

func (s *DeployService) extractZip(zipPath, destPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	if len(reader.File) > MaxFiles {
		return ErrTooManyFiles
	}

	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	// Check for common root directory in ZIP
	commonRoot := s.findCommonRoot(reader.File)

	for _, file := range reader.File {
		if err := s.extractFile(file, destPath, commonRoot); err != nil {
			return err
		}
	}

	return nil
}

func (s *DeployService) findCommonRoot(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}

	// Check if all files start with the same directory
	var commonRoot string
	for _, file := range files {
		name := file.Name
		if idx := strings.Index(name, "/"); idx > 0 {
			root := name[:idx+1]
			if commonRoot == "" {
				commonRoot = root
			} else if commonRoot != root {
				return "" // No common root
			}
		} else if !file.FileInfo().IsDir() {
			return "" // File at root level
		}
	}

	return commonRoot
}

func (s *DeployService) extractFile(file *zip.File, destPath, commonRoot string) error {
	// Get relative path, stripping common root if present
	name := file.Name
	if commonRoot != "" && strings.HasPrefix(name, commonRoot) {
		name = strings.TrimPrefix(name, commonRoot)
	}

	// Skip empty names (the root directory itself)
	if name == "" {
		return nil
	}

	// Validate path
	if err := s.validatePath(name); err != nil {
		return err
	}

	// Check for symlinks
	if file.Mode()&os.ModeSymlink != 0 {
		return ErrSymlinkDetected
	}

	// Build full path
	fullPath := filepath.Join(destPath, name)

	// Ensure the path is still within destPath (double-check after Join)
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(destPath)) {
		return ErrPathTraversal
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(fullPath, 0755)
	}

	// Check file size
	if file.UncompressedSize64 > MaxFileSize {
		return fmt.Errorf("%w: %s (%d bytes)", ErrFileTooLarge, name, file.UncompressedSize64)
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// Extract file
	destFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode().Perm())
	if err != nil {
		return err
	}
	defer destFile.Close()

	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Copy with size limit
	_, err = io.Copy(destFile, io.LimitReader(srcFile, MaxFileSize+1))
	return err
}

func (s *DeployService) validatePath(name string) error {
	// Check length
	if len(name) > MaxPathLength {
		return ErrInvalidPath
	}

	// Check for path traversal
	if strings.Contains(name, "..") {
		return ErrPathTraversal
	}

	// Check for absolute path
	if filepath.IsAbs(name) {
		return ErrPathTraversal
	}

	// Check for suspicious patterns
	cleanName := filepath.Clean(name)
	if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") {
		return ErrPathTraversal
	}

	// Check for null bytes
	if strings.ContainsRune(name, 0) {
		return ErrInvalidPath
	}

	return nil
}

func (s *DeployService) Rollback(siteID int64) error {
	sitePath := filepath.Join(s.config.Sites.Path, fmt.Sprintf("%d", siteID))
	publicPath := filepath.Join(sitePath, "public")
	publicPrevPath := filepath.Join(sitePath, "public_prev")

	// Check if previous version exists
	if _, err := os.Stat(publicPrevPath); os.IsNotExist(err) {
		return errors.New("no previous version available")
	}

	// Swap: public <-> public_prev
	tempPath := filepath.Join(sitePath, "public_temp")

	// Move current to temp
	if err := os.Rename(publicPath, tempPath); err != nil {
		return fmt.Errorf("move current to temp: %w", err)
	}

	// Move prev to current
	if err := os.Rename(publicPrevPath, publicPath); err != nil {
		// Restore
		os.Rename(tempPath, publicPath)
		return fmt.Errorf("restore previous: %w", err)
	}

	// Move temp to prev
	if err := os.Rename(tempPath, publicPrevPath); err != nil {
		// This is not critical, just log it
		os.RemoveAll(tempPath)
	}

	return nil
}

func (s *DeployService) ListDeploys(siteID int64, limit int) ([]*models.Deploy, error) {
	return s.deployRepo.ListBySite(siteID, limit)
}

func (s *DeployService) HasPreviousVersion(siteID int64) bool {
	sitePath := filepath.Join(s.config.Sites.Path, fmt.Sprintf("%d", siteID))
	publicPrevPath := filepath.Join(sitePath, "public_prev")
	_, err := os.Stat(publicPrevPath)
	return err == nil
}
