// This file contains the logic for translating HCL schema structs (from
// hcl_go) into the format-agnostic configuration model defined in the
// config package.

package hcl_adapter

import (
	"context"
	"fmt"

	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
)

// translateStep converts the HCL-specific step schema into the agnostic model.
func (l *Loader) translateStep(ctx context.Context, s *Step) *config.Step {
	logger := ctxlog.FromContext(ctx).With("step_runner", s.RunnerType, "step_name", s.Name)
	ctx = ctxlog.WithLogger(ctx, logger)

	logger.Debug("Translating HCL step to internal config model.")

	instancingMode := config.ModeSingular
	if isExprDefined(ctx, s.Count, "count") {
		logger.Debug("`count` attribute is defined. Marking step as instanced.")
		instancingMode = config.ModeInstanced
	} else {
		logger.Debug("`count` attribute is not defined. Marking step as singular.")
	}

	return &config.Step{
		RunnerType: s.RunnerType,
		Name:       s.Name,
		Count:      s.Count,
		Instancing: instancingMode,
		Arguments:  l.extractBodyAttributes(s.Arguments),
		Uses:       l.extractBodyAttributes(s.Uses),
		DependsOn:  s.DependsOn,
	}
}

// translateResource converts the HCL-specific resource schema into the agnostic model.
func (l *Loader) translateResource(s *Resource) *config.Resource {
	return &config.Resource{
		AssetType: s.AssetType,
		Name:      s.Name,
		Arguments: l.extractBodyAttributes(s.Arguments),
		DependsOn: s.DependsOn,
	}
}

// translateRunnerDefinition converts the HCL-specific runner schema into the agnostic model.
func (l *Loader) translateRunnerDefinition(ctx context.Context, s *RunnerDefinition) (*config.RunnerDefinition, error) {
	r := &config.RunnerDefinition{
		Type:        s.Type,
		Description: s.Description,
		Inputs:      make(map[string]*config.InputDefinition),
		Outputs:     make(map[string]*config.OutputDefinition),
		Uses:        make(map[string]*config.UsesDefinition),
	}
	if s.Lifecycle != nil {
		r.Lifecycle = &config.Lifecycle{OnRun: s.Lifecycle.OnRun}
	}

	for _, in := range s.Inputs {
		translatedInput, err := translateInputDefinition(ctx, in, "runner", s.Type)
		if err != nil {
			return nil, err
		}
		r.Inputs[in.Name] = translatedInput
	}

	for _, out := range s.Outputs {
		parsedType, err := typeExprToCtyType(ctx, out.Type)
		if err != nil {
			return nil, fmt.Errorf("in runner '%s', output '%s': %w", s.Type, out.Name, err)
		}
		r.Outputs[out.Name] = &config.OutputDefinition{
			Name:        out.Name,
			Type:        parsedType,
			Description: out.Description,
		}
	}

	for _, use := range s.Uses {
		r.Uses[use.LocalName] = &config.UsesDefinition{
			LocalName: use.LocalName,
			AssetType: use.AssetType,
		}
	}
	return r, nil
}

// translateAssetDefinition converts the HCL-specific asset schema into the agnostic model.
func (l *Loader) translateAssetDefinition(ctx context.Context, s *AssetDefinition) (*config.AssetDefinition, error) {
	a := &config.AssetDefinition{
		Type:        s.Type,
		Description: s.Description,
		Inputs:      make(map[string]*config.InputDefinition),
		Outputs:     make(map[string]*config.OutputDefinition),
	}
	if s.Lifecycle != nil {
		a.Lifecycle = &config.AssetLifecycle{Create: s.Lifecycle.Create, Destroy: s.Lifecycle.Destroy}
	}

	for _, in := range s.Inputs {
		translatedInput, err := translateInputDefinition(ctx, in, "asset", s.Type)
		if err != nil {
			return nil, err
		}
		a.Inputs[in.Name] = translatedInput
	}

	for _, out := range s.Outputs {
		parsedType, err := typeExprToCtyType(ctx, out.Type)
		if err != nil {
			return nil, fmt.Errorf("in asset '%s', output '%s': %w", s.Type, out.Name, err)
		}
		a.Outputs[out.Name] = &config.OutputDefinition{
			Name:        out.Name,
			Type:        parsedType,
			Description: out.Description,
		}
	}
	return a, nil
}
