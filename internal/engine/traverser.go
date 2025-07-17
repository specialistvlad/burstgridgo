package engine

import (
	"os"
	"path/filepath"
)

// FindHCLFiles scans a directory recursively for files with the .hcl extension.
func FindHCLFiles(rootDir string) ([]string, error) {
	var hclFiles []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".hcl" {
			hclFiles = append(hclFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return hclFiles, nil
}
