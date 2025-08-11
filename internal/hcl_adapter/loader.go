package hcl_adapter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
)

// Loader is the HCL-specific implementation of the config.Loader interface.
type Loader struct{}

// NewLoader creates a new HCL configuration loader.
func NewLoader() *Loader {
	return &Loader{}
}

// fileRoot is a struct used to decode all possible top-level blocks from any file.
type fileRoot struct {
	Runners   []*RunnerDefinition `hcl:"runner,block"`
	Assets    []*AssetDefinition  `hcl:"asset,block"`
	Steps     []*Step             `hcl:"step,block"`
	Resources []*Resource         `hcl:"resource,block"`
	Remain    hcl.Body            `hcl:",remain"`
}

// Load orchestrates the entire HCL configuration loading process. It is
// agnostic to the origin of the paths and parses any valid block from any file.
func (l *Loader) Load(ctx context.Context, paths ...string) (*config.Model, config.Converter, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("HCL loader started.", "path_count", len(paths))

	model := &config.Model{
		Runners: make(map[string]*config.RunnerDefinition),
		Assets:  make(map[string]*config.AssetDefinition),
		Grid:    &config.Grid{},
	}

	hclFiles, err := l.findAllHCLFiles(paths)
	if err != nil {
		return nil, nil, err
	}
	logger.Debug("Discovered HCL files.", "count", len(hclFiles))

	parser := hclparse.NewParser()

	for _, file := range hclFiles {
		hclFile, diags := parser.ParseHCLFile(file)
		if diags.HasErrors() {
			return nil, nil, fmt.Errorf("failed to parse HCL file %s: %w", file, diags)
		}

		var root fileRoot
		diags = gohcl.DecodeBody(hclFile.Body, nil, &root)
		if diags.HasErrors() {
			return nil, nil, fmt.Errorf("failed to decode HCL file %s: %w", file, diags)
		}

		// Translate and merge all discovered blocks into the model.
		for _, runner := range root.Runners {
			def, err := l.translateRunnerDefinition(ctx, runner)
			if err != nil {
				return nil, nil, err
			}
			model.Runners[def.Type] = def
		}
		for _, asset := range root.Assets {
			def, err := l.translateAssetDefinition(ctx, asset)
			if err != nil {
				return nil, nil, err
			}
			model.Assets[def.Type] = def
		}
		for _, step := range root.Steps {
			model.Grid.Steps = append(model.Grid.Steps, l.translateStep(ctx, step))
		}
		for _, resource := range root.Resources {
			model.Grid.Resources = append(model.Grid.Resources, l.translateResource(resource))
		}
	}

	logger.Debug("HCL loading complete.", "runners", len(model.Runners), "assets", len(model.Assets), "steps", len(model.Grid.Steps), "resources", len(model.Grid.Resources))
	return model, NewConverter(), nil
}

// findAllHCLFiles walks all given paths and returns a flat list of all .hcl files found.
func (l *Loader) findAllHCLFiles(paths []string) ([]string, error) {
	var allFiles []string
	seen := make(map[string]struct{})

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // It's not an error if a configured path doesn't exist.
			}
			return nil, fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if info.IsDir() {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && filepath.Ext(p) == ".hcl" {
					if _, wasSeen := seen[p]; !wasSeen {
						allFiles = append(allFiles, p)
						seen[p] = struct{}{}
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if filepath.Ext(path) == ".hcl" {
			if _, wasSeen := seen[path]; !wasSeen {
				allFiles = append(allFiles, path)
				seen[path] = struct{}{}
			}
		}
	}
	return allFiles, nil
}
