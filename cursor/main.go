package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"indexer/pkg/cache"
	"indexer/pkg/indexer"
	"indexer/pkg/search"
)

const usage = `Usage:
  indexer index <directory_path>  - Index files in the specified directory
  indexer search <keyword>        - Search for keyword in indexed files`

func main() {
	// Initialize components
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	cacheDir := filepath.Join(home, ".cache", "indexer")
	idx := indexer.NewIndex(runtime.NumCPU())
	cache := cache.NewCache(cacheDir)

	// Load cached data
	fmt.Println("Loading cache...")
	data, err := cache.Load()
	if err != nil {
		fmt.Printf("Warning: could not load cache: %v\n", err)
	} else {
		validFiles := 0
		for path, entry := range data {
			if _, err := os.Stat(path); err == nil {
				idx.GetFiles()[path] = entry
				validFiles++
			}
		}
		fmt.Printf("Loaded %d valid files from cache\n", validFiles)
	}

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	switch command {
	case "index":
		if flag.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "Error: index command requires a directory path")
			flag.Usage()
			os.Exit(1)
		}
		dirPath := flag.Arg(1)
		handleIndex(dirPath, idx, cache)

	case "search":
		if flag.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "Error: search command requires a keyword")
			flag.Usage()
			os.Exit(1)
		}
		keyword := flag.Arg(1)
		handleSearch(keyword, idx)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func handleIndex(dirPath string, idx *indexer.Index, cache *cache.Cache) {
	fmt.Printf("Indexing directory: %s\n", dirPath)

	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	if err := idx.IndexDirectory(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error indexing directory: %v\n", err)
		os.Exit(1)
	}

	// Save to cache
	fmt.Println("Saving to cache...")
	if err := cache.Save(idx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
	} else {
		fmt.Println("Cache saved successfully")
	}
}

func handleSearch(keyword string, idx *indexer.Index) {
	fmt.Printf("Searching for keyword: %s\n", keyword)

	results := search.Search(idx, keyword)
	if len(results) == 0 {
		fmt.Println("No matches found.")
		return
	}

	// Sort results by file path and line number
	sort.Slice(results, func(i, j int) bool {
		if results[i].FilePath == results[j].FilePath {
			return results[i].LineNumber < results[j].LineNumber
		}
		return results[i].FilePath < results[j].FilePath
	})

	fmt.Printf("\nFound matches in %d files:\n", len(results))
	currentFile := ""
	for _, result := range results {
		// Print file header when we switch to a new file
		if currentFile != result.FilePath {
			currentFile = result.FilePath
			relPath, err := filepath.Rel(".", currentFile)
			if err != nil {
				relPath = currentFile
			}
			fmt.Printf("\n%s:\n", relPath)
		}

		// Print the matching line with line number
		fmt.Printf("  %4d: %s\n", result.LineNumber, result.Line)
	}
	fmt.Println()
}
