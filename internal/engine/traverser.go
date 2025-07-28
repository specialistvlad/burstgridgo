package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vk/burstgridgo/internal/ctxlog"
)

// ResolveGridPath takes a path and returns a slice of all .hcl files found.
// If the path is a file, it returns a slice containing just that file.
// If the path is a directory, it recursively finds all .hcl files within it.
func ResolveGridPath(ctx context.Context, path string) ([]string, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Resolving grid path.", "path", path)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("grid path not found: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("error accessing path %s: %w", path, err)
	}

	if info.IsDir() {
		logger.Debug("Path is a directory, scanning for HCL files.", "directory", path)
		return findHCLFilesRecursive(ctx, path)
	}

	logger.Debug("Path is a single file.", "file", path)
	// If it's a file, ensure it has the .hcl extension.
	if filepath.Ext(path) != ".hcl" {
		return nil, fmt.Errorf("specified file is not an .hcl file: %s", path)
	}
	return []string{path}, nil
}

// findHCLFilesRecursive scans a directory recursively for files with the .hcl extension.
func findHCLFilesRecursive(ctx context.Context, rootDir string) ([]string, error) {
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
