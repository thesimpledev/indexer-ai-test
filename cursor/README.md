# Go Concurrent File Indexer CLI

This CLI application recursively scans and indexes files within a directory, providing fast, efficient keyword-based searching using Go's concurrency features.

## Overview

The application performs the following tasks:

1. **Indexing**: Recursively scans directories, concurrently reading and indexing file contents.
2. **Searching**: Quickly searches indexed content by keyword.
3. **Caching**: Persists indexed data to JSON to optimize repeated searches.

## CLI Usage

```shell
# Index a directory
$ indexer index <directory_path>
Indexed 312 files successfully.

# Search indexed files for a keyword
$ indexer search <keyword>
Found in:
 - path/to/file1.go:34
 - path/to/file2.txt:78
```

## Specifications

### Functional Requirements

- **Recursive Scanning**: Traverses directories recursively, managing permissions and symlinks gracefully.
- **Concurrency**: Uses goroutines and channels to concurrently read and index files for optimal performance.
- **Keyword-based Search**: Provides efficient keyword searches to quickly identify relevant files and line numbers.
- **Persistent Cache**: Stores indexing results in JSON to avoid redundant operations.

### Non-Functional Requirements

- **Robust Error Handling**: Continues indexing despite file-read errors, logging issues without interrupting the overall process.
- **Performance**: Efficient concurrency and memory management for quick indexing and searching.
- **Maintainability**: Clean, idiomatic Go code emphasizing readability.

## Constraints

- Only official Go standard libraries may be used (no third-party packages).
- Cross-platform compatibility (Linux, MacOS, Windows).

## Getting Started

Clone the repository, build the executable, and run commands:

```shell
go build -o indexer
./indexer index ./test-directory
./indexer search exampleKeyword
```

## Testing

Write tests using Go's built-in testing framework to validate core functionalities and ensure correctness.

---

This project serves as a robust example to evaluate AI coding tools (Cursor vs. Windsurf) for concurrency handling, filesystem operations, and Go programming workflow efficiency.

