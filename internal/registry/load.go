package registry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/fsutil"
	"github.com/specialistvlad/burstgridgo/internal/model"
)

func (reg *Registry) LoadGridsRecursively(ctx context.Context, modulesPath string) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Registry loading definitions from modules path...", "path", modulesPath)

	filePaths, err := fsutil.FindFilesByExtension(modulesPath, ".hcl")
	if err != nil {
		logger.Error("Failed to walk modules directory", "path", modulesPath, "error", err)
		return err
	}

	if len(filePaths) == 0 {
		logger.Warn("No .hcl module files found in path", "path", modulesPath)
		return nil
	}

	logger.Debug("Found HCL files to load", "files", filePaths)

	parser := hclparse.NewParser()

	for _, filePath := range filePaths {
		hclFile, diags := parser.ParseHCLFile(filePath)
		if diags.HasErrors() {
			return fmt.Errorf("failed to parse HCL file %s: %w", filePath, diags)
		}

		rn, err := model.NewRunner(ctx, hclFile, filePath)
		if err != nil {
			return fmt.Errorf("failed to process runner definition in %s: %w", filePath, err)
		}
		reg.runnersRegistry = append(reg.runnersRegistry, rn...)
		logger.Debug("Successfully loaded definitions from HCL file", "file", filePath)
	}

	logger.Info("Registry loaded successfully.", "runner_definitions_loaded", len(reg.runnersRegistry))
	return nil
}
