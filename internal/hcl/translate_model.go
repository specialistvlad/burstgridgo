// This file contains the logic for translating HCL schema structs (from
// hcl_schema.go) into the format-agnostic configuration model defined in the
// config package.

package hcl

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
)

// translateInputDefinition is a helper that processes a single HCL input
// block, handling its default value and type parsing.
func translateInputDefinition(ctx context.Context, in *schema.InputDefinition, ownerKind, ownerName string) (*config.InputDefinition, error) {
	var defaultVal *cty.Value
	var isOptional bool

	if in.Default != nil {
		val, diags := in.Default.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("invalid default value for input '%s' in %s '%s': %w", in.Name, ownerKind, ownerName, diags)
		}
		if !val.IsNull() {
			defaultVal = &val
			isOptional = true
		}
	}

	parsedType, err := typeExprToCtyType(ctx, in.Type)
	if err != nil {
		return nil, fmt.Errorf("in %s '%s', input '%s': %w", ownerKind, ownerName, in.Name, err)
	}

	return &config.InputDefinition{
		Name:        in.Name,
		Type:        parsedType,
		Description: in.Description,
		Default:     defaultVal,
		Optional:    isOptional,
	}, nil
}

// translateStep converts the HCL-specific step schema into the agnostic model.
func (l *Loader) translateStep(s *schema.Step) *config.Step {
	return &config.Step{
		RunnerType: s.RunnerType,
		Name:       s.Name,
		Arguments:  l.extractBodyAttributes(s.Arguments),
		Uses:       l.extractBodyAttributes(s.Uses),
		DependsOn:  s.DependsOn,
	}
}

// translateResource converts the HCL-specific resource schema into the agnostic model.
func (l *Loader) translateResource(s *schema.Resource) *config.Resource {
	return &config.Resource{
		AssetType: s.AssetType,
		Name:      s.Name,
		Arguments: l.extractBodyAttributes(s.Arguments),
		DependsOn: s.DependsOn,
	}
}

// translateRunnerDefinition converts the HCL-specific runner schema into the agnostic model.
func (l *Loader) translateRunnerDefinition(ctx context.Context, s *schema.RunnerDefinition) (*config.RunnerDefinition, error) {
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
func (l *Loader) translateAssetDefinition(ctx context.Context, s *schema.AssetDefinition) (*config.AssetDefinition, error) {
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

func (l *Loader) extractBodyAttributes(block interface{}) map[string]hcl.Expression {
	if block == nil {
		return nil
	}
	var body hcl.Body
	switch b := block.(type) {
	case *schema.StepArgs:
		if b == nil {
			return nil
		}
		body = b.Body
	case *schema.UsesBlock:
		if b == nil {
			return nil
		}
		body = b.Body
	default:
		return nil
	}
	if body == nil {
		return nil
	}
	attrs, _ := body.JustAttributes()
	if attrs == nil {
		return nil
	}
	exprMap := make(map[string]hcl.Expression)
	for name, attr := range attrs {
		exprMap[name] = attr.Expr
	}
	return exprMap
}
