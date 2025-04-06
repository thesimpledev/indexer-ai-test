package indexer

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Maximum size for the scanner buffer (16MB)
const maxScannerBufferSize = 16 * 1024 * 1024

// FileEntry represents an indexed file with its content information
type FileEntry struct {
	Path      string         `json:"path"`
	LineIndex map[int]string `json:"line_index"` // Maps line numbers to content
	Modified  int64          `json:"modified"`   // Last modified timestamp
}

// Index represents the main indexer that manages file scanning and indexing
type Index struct {
	mu      sync.RWMutex
	files   map[string]*FileEntry // Maps file paths to their entries
	workers int                   // Number of concurrent workers
	indexed uint64                // Number of files indexed
	skipped uint64                // Number of files skipped
}

// NewIndex creates a new indexer instance
func NewIndex(workers int) *Index {
	if workers <= 0 {
		workers = 1
	}
	return &Index{
		files:   make(map[string]*FileEntry),
		workers: workers,
	}
}

// IndexDirectory recursively indexes all files in the given directory
func (idx *Index) IndexDirectory(root string) error {
	fmt.Printf("Starting indexing of directory: %s\n", root)

	// Reset counters and clear existing files
	atomic.StoreUint64(&idx.indexed, 0)
	atomic.StoreUint64(&idx.skipped, 0)

	idx.mu.Lock()
	idx.files = make(map[string]*FileEntry)
	idx.mu.Unlock()

	// Create a channel to send file paths to workers
	paths := make(chan string)
	errors := make(chan error)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < idx.workers; i++ {
		wg.Add(1)
		go idx.worker(paths, errors, &wg)
	}

	// Start a goroutine to walk the directory
	go func() {
		defer close(paths)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Warning: error accessing %s: %v\n", path, err)
				return nil
			}
			if !info.IsDir() {
				// Skip binary files, hidden files, and very large files
				if isBinaryFile(path) || strings.HasPrefix(filepath.Base(path), ".") {
					fmt.Printf("Skipping file: %s (binary or hidden)\n", path)
					atomic.AddUint64(&idx.skipped, 1)
					return nil
				}
				if info.Size() > 100*1024*1024 {
					fmt.Printf("Skipping file: %s (too large: %.2f MB)\n", path, float64(info.Size())/(1024*1024))
					atomic.AddUint64(&idx.skipped, 1)
					return nil
				}
				paths <- path
			}
			return nil
		})
		if err != nil {
			errors <- fmt.Errorf("walk error: %w", err)
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(errors)
	}()

	// Collect any errors
	for err := range errors {
		if err != nil {
			fmt.Printf("Error during indexing: %v\n", err)
		}
	}

	// Print statistics
	indexed := atomic.LoadUint64(&idx.indexed)
	skipped := atomic.LoadUint64(&idx.skipped)

	idx.mu.RLock()
	totalFiles := len(idx.files)
	idx.mu.RUnlock()

	fmt.Printf("\nIndexing complete:\n")
	fmt.Printf("- Files processed: %d\n", indexed+skipped)
	fmt.Printf("- Files indexed: %d\n", indexed)
	fmt.Printf("- Files skipped: %d\n", skipped)
	fmt.Printf("- Total files in index: %d\n", totalFiles)

	// Print first few indexed files as debug info
	idx.mu.RLock()
	fmt.Println("\nFirst few indexed files:")
	count := 0
	for path, entry := range idx.files {
		if count >= 5 {
			break
		}
		lineCount := len(entry.LineIndex)
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		fmt.Printf("- %s (%d lines)\n", relPath, lineCount)
		// Print first few lines as sample
		if lineCount > 0 {
			fmt.Printf("  Sample lines:\n")
			sampleCount := 0
			for i := 1; i <= lineCount && sampleCount < 3; i++ {
				if line, ok := entry.LineIndex[i]; ok {
					fmt.Printf("    %d: %s\n", i, line)
					sampleCount++
				}
			}
		}
		count++
	}
	idx.mu.RUnlock()

	return nil
}

// worker processes files from the paths channel
func (idx *Index) worker(paths <-chan string, errors chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range paths {
		if err := idx.indexFile(path); err != nil {
			errors <- fmt.Errorf("error indexing %s: %w", path, err)
		} else {
			atomic.AddUint64(&idx.indexed, 1)
		}
	}
}

// isBinaryFile checks if a file is likely to be binary
func isBinaryFile(path string) bool {
	// Common binary file extensions
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".obj": true, ".o": true, ".a": true,
		".lib": true, ".pyc": true, ".class": true, ".jar": true,
		".war": true, ".ear": true, ".zip": true, ".tar": true,
		".gz": true, ".7z": true, ".rar": true, ".pdf": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".ico": true, ".mp3": true, ".mp4": true,
		".avi": true, ".mov": true, ".wmv": true, ".flv": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	return binaryExts[ext]
}

// indexFile indexes a single file
func (idx *Index) indexFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create a new entry for the file
	entry := &FileEntry{
		Path:      absPath,
		LineIndex: make(map[int]string),
		Modified:  info.ModTime().Unix(),
	}

	// Create a scanner with a larger buffer
	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxScannerBufferSize)
	scanner.Buffer(buf, maxScannerBufferSize)

	// Use custom split function to handle longer lines
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			// Return the line without the newline character
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	lineNum := 1
	for scanner.Scan() {
		entry.LineIndex[lineNum] = scanner.Text()
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning file: %w", err)
	}

	// Store the entry in the index
	idx.mu.Lock()
	idx.files[absPath] = entry
	idx.mu.Unlock()

	return nil
}

// GetFiles returns a copy of the indexed files map
func (idx *Index) GetFiles() map[string]*FileEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	files := make(map[string]*FileEntry, len(idx.files))
	for k, v := range idx.files {
		files[k] = v
	}

	fmt.Printf("GetFiles called - returning %d files\n", len(files))
	return files
}

// Stats returns the current indexing statistics
func (idx *Index) Stats() (indexed, skipped uint64) {
	return atomic.LoadUint64(&idx.indexed), atomic.LoadUint64(&idx.skipped)
}
