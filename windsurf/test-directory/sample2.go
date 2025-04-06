package main

import "fmt"

// This is a sample Go file to test our indexer
func main() {
	fmt.Println("Hello from the indexer test file")
	
	// This should be found when searching for 'keyword'
	keyword := "test"
	fmt.Printf("This is a %s of our indexer\n", keyword)
}
