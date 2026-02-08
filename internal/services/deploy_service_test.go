package services

import (
	"strings"
	"testing"
)

// makePathBytes creates a byte slice of 'a' characters for path testing
func makePathBytes(length int) []byte {
	return []byte(strings.Repeat("a", length))
}

func TestDeployService_validatePath(t *testing.T) {
	svc := &DeployService{}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "valid path",
			path:    "index.html",
			wantErr: nil,
		},
		{
			name:    "valid nested path",
			path:    "assets/css/style.css",
			wantErr: nil,
		},
		{
			name:    "path traversal with ..",
			path:    "../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path traversal in middle",
			path:    "foo/../../../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "absolute path",
			path:    "/etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path with null byte",
			path:    "file\x00.txt",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "too long path",
			path:    string(makePathBytes(MaxPathLength + 1)),
			wantErr: ErrInvalidPath,
		},
		{
			name:    "max length path",
			path:    string(makePathBytes(MaxPathLength)),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validatePath(tt.path)
			if err != tt.wantErr {
				t.Errorf("validatePath(%q) = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestDeployService_isZip(t *testing.T) {
	svc := &DeployService{}

	tests := []struct {
		filename string
		expected bool
	}{
		{"site.zip", true},
		{"site.ZIP", true},
		{"site.Zip", true},
		{"site.tar.gz", false},
		{"site.tgz", false},
		{"site.txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := svc.isZip(tt.filename); got != tt.expected {
				t.Errorf("isZip(%q) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestDeployService_isTarGz(t *testing.T) {
	svc := &DeployService{}

	tests := []struct {
		filename string
		expected bool
	}{
		{"site.tar.gz", true},
		{"site.TAR.GZ", true},
		{"site.tgz", true},
		{"site.TGZ", true},
		{"site.zip", false},
		{"site.tar", false},
		{"site.gz", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := svc.isTarGz(tt.filename); got != tt.expected {
				t.Errorf("isTarGz(%q) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if MaxArchiveSize != 100*1024*1024 {
		t.Errorf("MaxArchiveSize = %d, want %d", MaxArchiveSize, 100*1024*1024)
	}
	if MaxFileSize != 10*1024*1024 {
		t.Errorf("MaxFileSize = %d, want %d", MaxFileSize, 10*1024*1024)
	}
	if MaxFiles != 10000 {
		t.Errorf("MaxFiles = %d, want %d", MaxFiles, 10000)
	}
	if MaxPathLength != 500 {
		t.Errorf("MaxPathLength = %d, want %d", MaxPathLength, 500)
	}
}
