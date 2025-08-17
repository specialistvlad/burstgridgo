package testutil

import (
	"context"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
)

// RunHCLGridTest provides a simplified harness for testing the parsing of a
// single grid HCL string. It wraps the main integration test harness, providing
// dummy handlers that satisfy the parser for common runner types used in tests.
func RunHCLGridTest(t *testing.T, gridHCL string) (*HarnessResult, []*model.Step) {
	t.Helper()

	files := map[string]string{
		"grid/main.hcl": gridHCL,
	}

	// Create a generic "noop" handler for runner types used in parsing tests.
	mockHandler := &handlers.RegisteredHandler{
		Input: func() any { return new(struct{}) },
		Fn:    func(ctx context.Context, deps, input any) (any, error) { return nil, nil },
	}
	handlerStorage := handlers.New()

	// Runner manifests link a runner type (e.g. "print") to a Go handler name (e.g. "OnRunPrint").
	// Since we are only testing the grid parser, we can assume a few common mappings exist.
	handlerStorage.RegisterHandler("OnRunPrint", mockHandler)
	handlerStorage.RegisterHandler("OnRunConfig", mockHandler)
	handlerStorage.RegisterHandler("OnRunTest", mockHandler)
	handlerStorage.RegisterHandler("OnRunNoop", mockHandler)

	result := RunIntegrationTest(t, files, handlerStorage)

	if result.App != nil && result.App.Grid() != nil {
		return result, result.App.Grid().Steps
	}

	return result, nil
}
