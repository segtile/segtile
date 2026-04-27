package util

import (
	"os"
	"path/filepath"
	"sort"
)

func createAndOpen(filePath string) *os.File {

	// Ensure the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// Create the file if it does not exist
		_, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
	}

	// Open the file for reading and writing
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	return file
}

func ListFiles(dir string) ([]string, error) {

	// Read all files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// Construct the full file path
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == ".DS_Store" || entry.Name() == "Thumbs.db" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	// Sort the files alphabetically
	sort.Strings(files)

	return files, nil
}
