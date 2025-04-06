package cache

import (
	"encoding/json"
	"os"
	"path/filepath"

	"indexer/pkg/indexer"
)

const defaultCacheFile = ".indexer_cache.json"

// Cache handles persistent storage of indexed data
type Cache struct {
	filePath string
}

// NewCache creates a new cache instance
func NewCache(cacheDir string) *Cache {
	return &Cache{
		filePath: filepath.Join(cacheDir, defaultCacheFile),
	}
}

// Save persists the index data to disk
func (c *Cache) Save(idx *indexer.Index) error {
	data := idx.GetFiles()

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(c.filePath), 0755); err != nil {
		return err
	}

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(c.filePath, jsonData, 0644)
}

// Load reads the index data from disk
func (c *Cache) Load() (map[string]*indexer.FileEntry, error) {
	data := make(map[string]*indexer.FileEntry)

	// Read file
	jsonData, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, err
	}

	// Unmarshal JSON
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	return data, nil
}
