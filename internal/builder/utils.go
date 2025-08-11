package builder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// formatTraversal converts an hcl.Traversal to a human-readable string for logging.
func formatTraversal(t hcl.Traversal) string {
	var sb strings.Builder
	for i, part := range t {
		switch p := part.(type) {
		case hcl.TraverseRoot:
			sb.WriteString(p.Name)
		case hcl.TraverseAttr:
			sb.WriteRune('.')
			sb.WriteString(p.Name)
		case hcl.TraverseIndex:
			sb.WriteRune('[')
			if p.Key.Type() == cty.String {
				sb.WriteString(fmt.Sprintf("%q", p.Key.AsString()))
			} else if p.Key.Type() == cty.Number {
				bf := p.Key.AsBigFloat()
				sb.WriteString(bf.Text('f', -1))
			} else {
				sb.WriteString("...")
			}
			sb.WriteRune(']')
		default:
			if i > 0 {
				sb.WriteRune('.')
			}
			sb.WriteString("?")
		}
	}
	return sb.String()
}

// depAddress represents a parsed dependency reference.
// Index is -1 if not specified (shorthand).
type depAddress struct {
	Name  string
	Index int
}

// depAddrRegex is used to parse addresses like "name" or "name[index]".
var depAddrRegex = regexp.MustCompile(`^([a-zA-Z0-9_.-]+)(?:\[(\d+)\])?$`)

// parseDepAddress parses a raw dependency string into its name and optional index.
func parseDepAddress(addr string) (*depAddress, error) {
	matches := depAddrRegex.FindStringSubmatch(addr)
	if matches == nil {
		return nil, fmt.Errorf("invalid dependency address format: %q", addr)
	}

	name := matches[1]
	index := -1 // Default to -1 for shorthand reference
	if matches[2] != "" {
		var err error
		index, err = strconv.Atoi(matches[2])
		if err != nil {
			// This should be unreachable due to the regex \d+
			return nil, fmt.Errorf("internal error: failed to parse index from %q", addr)
		}
	}
	return &depAddress{Name: name, Index: index}, nil
}

// validateOutputReference checks if a reference to a step's output is valid.
func validateOutputReference(traversal hcl.Traversal, depNode *node.Node, r *registry.Registry) error {
	if depNode.Type != node.StepNode || len(traversal) < 5 {
		return nil // Not a step output reference we need to validate.
	}

	outputNameAttr, ok := traversal[4].(hcl.TraverseAttr)
	if !ok {
		return nil // Malformed traversal.
	}
	outputName := outputNameAttr.Name

	runnerDef, ok := r.DefinitionRegistry[depNode.StepConfig.RunnerType]
	if !ok {
		return fmt.Errorf("internal error: could not find definition for runner type %s", depNode.StepConfig.RunnerType)
	}

	if _, ok := runnerDef.Outputs[outputName]; ok {
		return nil // Found a valid declaration.
	}

	return fmt.Errorf("reference to undeclared output %q on step %q", outputName, depNode.ID())
}

// expandStep analyzes a step's instancing configuration. It returns a slice of
// step configs and a boolean indicating if the step is a dynamic placeholder.
func expandStep(s *config.Step) (steps []*config.Step, isPlaceholder bool) {
	if s.Instancing != config.ModeInstanced || s.Count == nil {
		return []*config.Step{s}, false
	}

	val, diags := s.Count.Value(nil)
	if diags.HasErrors() || !val.IsKnown() {
		return []*config.Step{s}, true
	}

	if val.Type() != cty.Number {
		return []*config.Step{s}, false
	}

	countBf, _ := val.AsBigFloat().Int64()
	count := int(countBf)

	if count <= 0 {
		return []*config.Step{}, false
	}

	instances := make([]*config.Step, count)
	for i := 0; i < count; i++ {
		instanceConf := *s
		instances[i] = &instanceConf
	}
	return instances, false
}
