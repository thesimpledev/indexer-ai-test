package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SearchResult represents a single search result with file path and line number
type SearchResult struct {
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
}

// FileIndex represents the index data for a single file
type FileIndex struct {
	Path     string            `json:"path"`
	LineMap  map[int]string    `json:"lineMap"`
	Modified int64             `json:"modified"`
}

// Index represents the entire index data structure
type Index struct {
	Files map[string]*FileIndex `json:"files"`
}

// Indexer handles file indexing and searching operations
type Indexer struct {
	index         Index
	indexFilePath string
	mutex         sync.RWMutex
}

// NewIndexer creates a new instance of Indexer
func NewIndexer() *Indexer {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	indexFilePath := filepath.Join(homeDir, ".indexer_data.json")
	
	return &Indexer{
		index: Index{
			Files: make(map[string]*FileIndex),
		},
		indexFilePath: indexFilePath,
		mutex:         sync.RWMutex{},
	}
}

// IndexDirectory recursively indexes all files in the specified directory
func (idx *Indexer) IndexDirectory(rootDir string) (int, error) {
	filesChan := make(chan string)
	errorsChan := make(chan error)
	resultsChan := make(chan *FileIndex)
	var wg sync.WaitGroup

	// Start worker goroutines
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range filesChan {
				fileIndex, err := idx.indexFile(filePath)
				if err != nil {
					errorsChan <- fmt.Errorf("error indexing %s: %w", filePath, err)
					continue
				}
				resultsChan <- fileIndex
			}
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	done := make(chan struct{})
	go func() {
		for fileIndex := range resultsChan {
			idx.mutex.Lock()
			idx.index.Files[fileIndex.Path] = fileIndex
			idx.mutex.Unlock()
		}
		close(done)
	}()

	// Walk the directory tree
	go func() {
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Use utility functions to determine if file should be indexed
			if ShouldIndexFile(path) && IsTextFile(path) {
				filesChan <- path
			}
			
			return nil
		})

		close(filesChan)
		if err != nil {
			errorsChan <- err
		}
		close(errorsChan)
	}()

	// Process any errors
	var indexingErrors []error
	for err := range errorsChan {
		indexingErrors = append(indexingErrors, err)
	}

	// Wait for result collection to finish
	<-done

	if len(indexingErrors) > 0 {
		return len(idx.index.Files), fmt.Errorf("encountered %d errors during indexing", len(indexingErrors))
	}

	return len(idx.index.Files), nil
}

// indexFile indexes a single file
func (idx *Indexer) indexFile(filePath string) (*FileIndex, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileIndex := &FileIndex{
		Path:     filePath,
		LineMap:  make(map[int]string),
		Modified: info.ModTime().Unix(),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		fileIndex.LineMap[lineNum] = scanner.Text()
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return fileIndex, nil
}

// Search finds all occurrences of a keyword in the indexed files
func (idx *Indexer) Search(keyword string) ([]SearchResult, error) {
	keyword = strings.ToLower(keyword)
	var results []SearchResult

	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	for _, fileIndex := range idx.index.Files {
		for lineNum, lineText := range fileIndex.LineMap {
			if strings.Contains(strings.ToLower(lineText), keyword) {
				results = append(results, SearchResult{
					FilePath:   fileIndex.Path,
					LineNumber: lineNum,
				})
			}
		}
	}

	return results, nil
}

// SaveIndex persists the index to a file
func (idx *Indexer) SaveIndex() error {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	file, err := os.Create(idx.indexFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(idx.index)
}

// LoadIndex loads the index from a file
func (idx *Indexer) LoadIndex() error {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	file, err := os.Open(idx.indexFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&idx.index)
}
