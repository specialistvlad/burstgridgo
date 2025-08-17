// Package bggoexpr provides a container for collecting and analyzing HCL expressions.
package bggoexpr

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
)

// Container is a thread-safe helper that gathers HCL expressions and provides
// analysis results, such as variable references and function calls.
type Container struct {
	// analyzeOnce ensures the extraction logic runs exactly once.
	analyzeOnce sync.Once

	mu          sync.RWMutex // Use RWMutex for better read performance
	expressions []hcl.Expression

	// Caching fields for analysis results
	references      []hcl.Traversal
	calledFunctions []string
}

// NewContainer creates a new, empty expression container.
func NewContainer() *Container {
	return &Container{}
}

// Add adds one or more expressions to the container for analysis.
// It safely ignores any nil expressions.
func (c *Container) Add(exprs ...hcl.Expression) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Adding new expressions requires resetting the sync.Once so analysis can run again.
	// NOTE: This is safe as long as Add is not called concurrently with the getters.
	// In our use case, all Adds happen during the initial parsing phase, which is single-threaded.
	c.analyzeOnce = sync.Once{}

	for _, expr := range exprs {
		if expr != nil {
			c.expressions = append(c.expressions, expr)
		}
	}
}

// analyze performs the dependency extraction. It's guaranteed to run only once
// for a given set of expressions due to sync.Once.
func (c *Container) analyze() {
	c.analyzeOnce.Do(func() {
		// The actual extraction doesn't need a lock because Do() is atomic.
		// However, we need to read c.expressions safely.
		c.mu.RLock()
		refs, funcs := extractReferencesAndFunctions(c.expressions...)
		c.mu.RUnlock()

		// But we need a write lock to update the result fields.
		c.mu.Lock()
		c.references = refs
		c.calledFunctions = funcs
		c.mu.Unlock()
	})
}

// References returns all unique variable traversals found in the expressions.
func (c *Container) References() []hcl.Traversal {
	c.analyze()
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.references
}

// CalledFunctions returns all unique function calls found in the expressions.
func (c *Container) CalledFunctions() []string {
	c.analyze()
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.calledFunctions
}
