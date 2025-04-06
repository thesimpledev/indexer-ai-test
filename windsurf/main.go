package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		handleIndex()
	case "search":
		handleSearch()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  indexer index <directory_path>  - Index files in the specified directory")
	fmt.Println("  indexer search <keyword>        - Search for keyword in indexed files")
}

func handleIndex() {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	indexCmd.Parse(os.Args[2:])

	if indexCmd.NArg() < 1 {
		fmt.Println("Error: directory path required")
		fmt.Println("Usage: indexer index <directory_path>")
		os.Exit(1)
	}

	dirPath := indexCmd.Arg(0)
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		os.Exit(1)
	}

	indexer := NewIndexer()
	count, err := indexer.IndexDirectory(dirPath)
	if err != nil {
		fmt.Printf("Error during indexing: %v\n", err)
		os.Exit(1)
	}

	err = indexer.SaveIndex()
	if err != nil {
		fmt.Printf("Error saving index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Indexed %d files successfully.\n", count)
}

func handleSearch() {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	searchCmd.Parse(os.Args[2:])

	if searchCmd.NArg() < 1 {
		fmt.Println("Error: search keyword required")
		fmt.Println("Usage: indexer search <keyword>")
		os.Exit(1)
	}

	keyword := searchCmd.Arg(0)

	indexer := NewIndexer()
	err := indexer.LoadIndex()
	if err != nil {
		fmt.Printf("Error loading index: %v\n", err)
		fmt.Println("Have you indexed any directories yet?")
		os.Exit(1)
	}

	results, err := indexer.Search(keyword)
	if err != nil {
		fmt.Printf("Error during search: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No results found for:", keyword)
		return
	}

	fmt.Println("Found in:")
	for _, result := range results {
		fmt.Printf(" - %s:%d\n", result.FilePath, result.LineNumber)
	}
}
