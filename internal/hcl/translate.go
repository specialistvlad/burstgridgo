package hcl

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
)

// typeExprToCtyType converts an HCL type expression into its cty.Type equivalent.
func typeExprToCtyType(ctx context.Context, expr hcl.Expression) (cty.Type, error) {
	logger := ctxlog.FromContext(ctx)

	if expr == nil {
		logger.Debug("Type expression is nil, defaulting to any.")
		return cty.DynamicPseudoType, nil
	}

	// Using a type switch is the correct way to handle the various concrete
	// expression types that implement the hcl.Expression interface.
	switch v := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		logger.Debug("Parsing type expression as a function call.", "call", v.Name)
		if len(v.Args) != 1 {
			return cty.DynamicPseudoType, fmt.Errorf("type constructors (list, map, set) require exactly one argument, got %d", len(v.Args))
		}

		// Recursively parse the inner type.
		elementType, err := typeExprToCtyType(ctx, v.Args[0])
		if err != nil {
			return cty.DynamicPseudoType, err
		}
		if elementType == cty.DynamicPseudoType {
			return cty.DynamicPseudoType, fmt.Errorf("collection types cannot contain type 'any'")
		}
		logger.Debug("Parsed collection element type.", "type", elementType.FriendlyName())

		switch v.Name {
		case "list":
			return cty.List(elementType), nil
		case "map":
			return cty.Map(elementType), nil
		case "set":
			return cty.Set(elementType), nil
		default:
			return cty.DynamicPseudoType, fmt.Errorf("unknown type constructor function %q", v.Name)
		}

	case *hclsyntax.ScopeTraversalExpr:
		// This handles primitive type identifiers like `string` or `number`.
		if len(v.Traversal) != 1 {
			return cty.DynamicPseudoType, fmt.Errorf("invalid type keyword: traversal path is not a single identifier")
		}
		rootName := v.Traversal.RootName()
		logger.Debug("Parsing type expression as a primitive.", "keyword", rootName)
		switch rootName {
		case "string":
			return cty.String, nil
		case "number":
			return cty.Number, nil
		case "bool":
			return cty.Bool, nil
		case "any":
			return cty.DynamicPseudoType, nil
		default:
			return cty.DynamicPseudoType, fmt.Errorf("unknown primitive type %q", rootName)
		}

	default:
		// Fallback for any other kind of expression.
		return cty.DynamicPseudoType, fmt.Errorf("unsupported expression for type definition: %T", v)
	}
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
		var defaultVal *cty.Value
		var isOptional bool

		if in.Default != nil {
			val, diags := in.Default.Value(nil)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid default value for input '%s' in runner '%s': %w", in.Name, s.Type, diags)
			}
			if !val.IsNull() {
				defaultVal = &val
				isOptional = true
			}
		}

		parsedType, err := typeExprToCtyType(ctx, in.Type)
		if err != nil {
			// Provide more context for a parsing error.
			return nil, fmt.Errorf("in runner '%s', input '%s': %w", s.Type, in.Name, err)
		}

		r.Inputs[in.Name] = &config.InputDefinition{
			Name:        in.Name,
			Type:        parsedType,
			Description: in.Description,
			Default:     defaultVal,
			Optional:    isOptional,
		}
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
		var defaultVal *cty.Value
		var isOptional bool

		if in.Default != nil {
			val, diags := in.Default.Value(nil)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid default value for input '%s' in asset '%s': %w", in.Name, s.Type, diags)
			}
			if !val.IsNull() {
				defaultVal = &val
				isOptional = true
			}
		}

		parsedType, err := typeExprToCtyType(ctx, in.Type)
		if err != nil {
			return nil, fmt.Errorf("in asset '%s', input '%s': %w", s.Type, in.Name, err)
		}

		a.Inputs[in.Name] = &config.InputDefinition{
			Name:        in.Name,
			Type:        parsedType,
			Description: in.Description,
			Default:     defaultVal,
			Optional:    isOptional,
		}
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
