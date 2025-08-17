// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the Grid structure, which is the root container for all
// configuration loaded from a user's .hcl files.
//
// Why have a Grid?
//
// The Grid serves as the top-level aggregator for the entire execution graph.
// In a real-world scenario, a user might split their configuration across many
// files and directories. The purpose of the Grid and its loading functions is to
// discover all these disparate 'step' blocks and consolidate them into a single,
// unified view.
//
// By aggregating everything into one place, we enable workspace-wide analysis.
// The graph builder can operate on the complete set of steps within the Grid to
// resolve dependencies that span across different files, which is a critical
// feature for building complex workflows.
package model

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/fsutil"
)

// Grid represents the user's execution graph definition.
type Grid struct {
	Steps []*Step
}

// NewGrid creates and returns an initialized Grid.
func NewGrid() *Grid {
	return &Grid{
		Steps: []*Step{},
	}
}

// hclGridFile represents the top-level structure of a grid file for decoding.
type hclGridFile struct {
	Steps     []*hclStep          `hcl:"step,block"`
	Locals    []*hclLocalsBlock   `hcl:"locals,block"`
	Variables []*hclVariableBlock `hcl:"variable,block"`
}

// newGridFromHCL parses a single HCL file and returns the Steps found within it.
func newGridFromHCL(filePath string, parser *hclparse.Parser) ([]*Step, error) {
	hclFile, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file %s: %w", filePath, diags)
	}

	var parsedFile hclGridFile
	diags = gohcl.DecodeBody(hclFile.Body, nil, &parsedFile)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL file %s: %w", filePath, diags)
	}

	// For now, we only parse steps. The presence of the `Locals` and `Variables`
	// fields in the struct is enough to prevent the parser from erroring.

	steps := make([]*Step, 0, len(parsedFile.Steps))
	for _, parsedStep := range parsedFile.Steps {
		step, stepDiags := NewStepFromHCL(parsedStep, filePath)
		if stepDiags.HasErrors() {
			return nil, fmt.Errorf("error parsing step in file %s: %w", filePath, stepDiags)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// LoadGridsRecursively finds and parses all HCL files in a given path into a Grid model.
func LoadGridsRecursively(ctx context.Context, gridPath string) (*Grid, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Loading grid from path", "path", gridPath)

	files, err := fsutil.FindFilesByExtension(gridPath, ".hcl")
	if err != nil {
		return nil, fmt.Errorf("failed to find grid files in %s: %w", gridPath, err)
	}

	grid := NewGrid()
	if len(files) == 0 {
		logger.Warn("No .hcl grid files found in path, returning empty grid", "path", gridPath)
		return grid, nil
	}

	parser := hclparse.NewParser()
	for _, file := range files {
		steps, err := newGridFromHCL(file, parser)
		if err != nil {
			return nil, err
		}
		grid.Steps = append(grid.Steps, steps...)
	}

	return grid, nil
}
