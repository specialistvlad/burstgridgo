// Package fsutil provides file system utility functions.
package fsutil

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// FindFilesByExtension recursively searches the given root path for all files ending
// with the specified extension. It returns a slice of their full paths.
func FindFilesByExtension(rootPath string, extension string) ([]string, error) {
	if extension == "" {
		panic("extension must not be empty")
	}

	var files []string
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), extension) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
