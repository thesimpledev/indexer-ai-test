package main

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultExcludedDirs is a list of directories that are excluded from indexing by default
var DefaultExcludedDirs = []string{
	".git", ".svn", "node_modules", "vendor", "bin", "obj",
}

// DefaultExcludedExtensions is a list of file extensions that are excluded from indexing by default
var DefaultExcludedExtensions = []string{
	".exe", ".dll", ".so", ".dylib", ".o", ".a", ".out",
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".svg",
	".mp3", ".mp4", ".avi", ".mkv", ".mov", ".flv", ".wmv",
	".zip", ".tar", ".gz", ".rar", ".7z", ".jar", ".war",
	".class", ".pyc", ".pyo", ".obj",
}

// ShouldIndexFile determines if a file should be indexed based on path and extension
func ShouldIndexFile(path string) bool {
	// Check if the file exists and is readable
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}

	// Check file size (skip files larger than 10MB)
	if info.Size() > 10*1024*1024 {
		return false
	}

	// Check if file is in an excluded directory
	dirPath := filepath.Dir(path)
	for _, excludedDir := range DefaultExcludedDirs {
		if strings.Contains(dirPath, excludedDir) {
			return false
		}
	}

	// Check if file has an excluded extension
	ext := strings.ToLower(filepath.Ext(path))
	for _, excludedExt := range DefaultExcludedExtensions {
		if ext == excludedExt {
			return false
		}
	}

	return true
}

// IsTextFile attempts to determine if a file is a text file
func IsTextFile(path string) bool {
	// Common text file extensions
	textExtensions := []string{
		".txt", ".md", ".json", ".xml", ".html", ".htm", ".css", ".js",
		".go", ".py", ".java", ".c", ".cpp", ".h", ".hpp", ".cs", ".php",
		".rb", ".pl", ".sh", ".bat", ".ps1", ".yaml", ".yml", ".toml",
		".ini", ".cfg", ".conf", ".log", ".csv", ".tsv",
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	// For files without recognized extensions, we could check the content
	// to determine if it's text, but for simplicity we'll just rely on extensions
	return false
}
