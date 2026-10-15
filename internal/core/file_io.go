package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ListFiles(dir string, imagesOnly bool) ([]string, error) {

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

		if imagesOnly {
			// Skip files that are not images based on their extension
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".tiff" && ext != ".tif" {
				continue
			}
		}

		files = append(files, filepath.Join(dir, entry.Name()))
	}

	// Sort the files alphabetically
	sort.Strings(files)

	return files, nil
}
