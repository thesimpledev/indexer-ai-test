package search

import (
	"fmt"
	"strings"
	"sync"

	"indexer/pkg/indexer"
)

// SearchResult represents a single match in a file
type SearchResult struct {
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
	MatchCount int    `json:"match_count"`
}

// Search performs a concurrent search across all indexed files
func Search(idx *indexer.Index, keyword string) []SearchResult {
	files := idx.GetFiles()
	fmt.Printf("Searching through %d indexed files\n", len(files))

	results := make([]SearchResult, 0)
	resultsChan := make(chan SearchResult)
	var wg sync.WaitGroup

	// Convert keyword to lowercase for case-insensitive search
	keyword = strings.ToLower(keyword)

	// Start a worker for each file
	for path, entry := range files {
		wg.Add(1)
		// Create local variables to avoid race condition
		filePath := path
		fileEntry := entry
		go func() {
			defer wg.Done()
			searchFile(filePath, fileEntry, keyword, resultsChan)
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	matchCount := 0
	fileCount := 0
	filesSeen := make(map[string]bool)

	for result := range resultsChan {
		if result.MatchCount > 0 {
			results = append(results, result)
			matchCount += result.MatchCount
			if !filesSeen[result.FilePath] {
				fileCount++
				filesSeen[result.FilePath] = true
			}
		}
	}

	fmt.Printf("Found %d matches in %d files\n", matchCount, fileCount)
	return results
}

// searchFile searches for the keyword in a single file
func searchFile(path string, entry *indexer.FileEntry, keyword string, results chan<- SearchResult) {
	for lineNum, line := range entry.LineIndex {
		lowerLine := strings.ToLower(line)
		count := strings.Count(lowerLine, keyword)
		if count > 0 {
			results <- SearchResult{
				FilePath:   path,
				LineNumber: lineNum,
				Line:       line,
				MatchCount: count,
			}
		}
	}
}
