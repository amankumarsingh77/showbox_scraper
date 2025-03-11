package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

// Function to find index of title
func findTitleIndex(items []map[string]interface{}, targetTitle string) int {
	for i, item := range items {
		if title, ok := item["title"].(string); ok && title == targetTitle {
			return i
		}
	}
	return -1 // Return -1 if title not found
}

func main() {
	// Read JSON file
	data, err := ioutil.ReadFile("tv_final.json") // Replace with your file name
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Define a slice of maps to store JSON data
	var items []map[string]interface{}

	// Unmarshal JSON into slice
	err = json.Unmarshal(data, &items)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Define the title to search for
	targetTitle := "Irina: The Vampire Cosmonaut"

	// Find index of the title
	index := findTitleIndex(items, targetTitle)

	// Print the result
	if index != -1 {
		fmt.Printf("Title '%s' found at index: %d\n", targetTitle, index)
	} else {
		fmt.Printf("Title '%s' not found in the JSON file\n", targetTitle)
	}
}
