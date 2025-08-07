package dag

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/registry"
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
			// The first part of a traversal is TraverseRoot, so any attr
			// will have something before it.
			sb.WriteRune('.')
			sb.WriteString(p.Name)
		case hcl.TraverseIndex:
			sb.WriteRune('[')
			if p.Key.Type() == cty.String {
				// Format strings with quotes
				sb.WriteString(fmt.Sprintf("%q", p.Key.AsString()))
			} else if p.Key.Type() == cty.Number {
				// Format numbers directly
				bf := p.Key.AsBigFloat()
				sb.WriteString(bf.Text('f', -1))
			} else {
				// Fallback for any other complex key type
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

// detectCycles checks for circular dependencies in the graph using DFS.
func (g *Graph) detectCycles() error {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var visit func(node *Node) error
	visit = func(node *Node) error {
		visiting[node.ID] = true
		for _, dep := range node.Deps {
			if visiting[dep.ID] {
				return fmt.Errorf("cycle detected involving '%s'", dep.ID)
			}
			if !visited[dep.ID] {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		delete(visiting, node.ID)
		visited[node.ID] = true
		return nil
	}

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			if err := visit(node); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateOutputReference checks if a reference to a step's output is valid.
func validateOutputReference(traversal hcl.Traversal, depNode *Node, r *registry.Registry) error {
	if depNode.Type != StepNode || len(traversal) < 5 {
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

	return fmt.Errorf("reference to undeclared output %q on step %q", outputName, depNode.ID)
}

func expandStep(s *config.Step) []*config.Step {
	// If not instanced, return immediately.
	if s.Instancing != config.ModeInstanced || s.Count == nil {
		return []*config.Step{s}
	}

	// Attempt to statically evaluate the expression.
	val, diags := s.Count.Value(nil) // Use nil context for static eval.
	if diags.HasErrors() || !val.IsKnown() {
		// If it's not a static value, we can't expand it here.
		// Dynamic evaluation will be handled in a later step.
		// For now, treat it as a single instance to keep the DAG valid.
		return []*config.Step{s}
	}

	if val.Type() != cty.Number {
		// Invalid type for count, treat as single. Proper validation later.
		return []*config.Step{s}
	}

	countBf, _ := val.AsBigFloat().Int64()
	count := int(countBf)

	if count <= 0 {
		// Count of 0 or negative means no instances are created.
		return []*config.Step{}
	}

	// Create 'count' copies of the step config.
	instances := make([]*config.Step, count)
	for i := 0; i < count; i++ {
		// Create a shallow copy for each instance. This is safe because the
		// contents are just values or pointers to immutable expressions.
		instanceConf := *s
		instances[i] = &instanceConf
	}
	return instances
}
