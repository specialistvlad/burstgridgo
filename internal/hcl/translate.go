package hcl

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
)

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
func (l *Loader) translateRunnerDefinition(ctx context.Context, s *schema.RunnerDefinition) *config.RunnerDefinition {
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
		var defaultVal *cty.Value
		var isOptional bool

		if in.Default != nil {
			val, diags := in.Default.Value(nil)
			// A default is only valid if it evaluates without error AND is not null.
			if !diags.HasErrors() && !val.IsNull() {
				defaultVal = &val
				isOptional = true // It's optional because a valid default exists.
			}
		}

		r.Inputs[in.Name] = &config.InputDefinition{
			Name:        in.Name,
			Type:        cty.DynamicPseudoType, // Placeholder until ADR-009
			Description: in.Description,
			Default:     defaultVal,
			Optional:    isOptional, // This is now set correctly.
		}
	}
	for _, out := range s.Outputs {
		r.Outputs[out.Name] = &config.OutputDefinition{
			Name:        out.Name,
			Type:        cty.DynamicPseudoType, // Placeholder until ADR-009
			Description: out.Description,
		}
	}
	for _, use := range s.Uses {
		r.Uses[use.LocalName] = &config.UsesDefinition{
			LocalName: use.LocalName,
			AssetType: use.AssetType,
		}
	}
	return r
}

// translateAssetDefinition converts the HCL-specific asset schema into the agnostic model.
func (l *Loader) translateAssetDefinition(ctx context.Context, s *schema.AssetDefinition) *config.AssetDefinition {
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
		var defaultVal *cty.Value
		var isOptional bool

		if in.Default != nil {
			val, diags := in.Default.Value(nil)
			// A default is only valid if it evaluates without error AND is not null.
			if !diags.HasErrors() && !val.IsNull() {
				defaultVal = &val
				isOptional = true // It's optional because a valid default exists.
			}
		}

		a.Inputs[in.Name] = &config.InputDefinition{
			Name:        in.Name,
			Type:        cty.DynamicPseudoType,
			Description: in.Description,
			Default:     defaultVal,
			Optional:    isOptional,
		}
	}
	for _, out := range s.Outputs {
		a.Outputs[out.Name] = &config.OutputDefinition{
			Name:        out.Name,
			Type:        cty.DynamicPseudoType,
			Description: out.Description,
		}
	}
	return a
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
