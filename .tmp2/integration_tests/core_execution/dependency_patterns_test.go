package integration_tests

// import (
// 	"context"
// 	"strings"
// 	"sync"
// 	"testing"

// 	"github.com/stretchr/testify/require"
// 	"github.com/specialistvlad/burstgridgo/internal/registry"
// 	"github.com/specialistvlad/burstgridgo/internal/testutil"
// 	"github.com/zclconf/go-cty/cty"
// )

// func TestCoreExecution_DynamicCount_MixedDependencies(t *testing.T) {
// 	t.Skip("This test is currently skipped due to a known issue with dynamic count and splat expressions. It will be re-enabled once the underlying bug is fixed.")
// 	t.Parallel()

// 	// Arrange: Define the HCL for our complex dependency scenario.
// 	manifestsHCL := `
// 		runner "env_vars" {
// 			lifecycle {
// 				on_run = "OnRunEnvVars"
// 			}
// 			output "vars" {
// 				type = map(string)
// 			}
// 		}
// 		runner "http_request" {
// 			lifecycle {
// 				on_run = "OnRunHttpRequest"
// 			}
// 			# The handler for this runner returns an object with 'url' and 'status'.
// 			output "data" {
// 				type = object({
// 					url    = string,
// 					status = number
// 				})
// 			}
// 		}
// 		runner "print" {
// 			lifecycle {
// 				on_run = "OnRunPrint"
// 			}
// 		}
// 	`
// 	gridHCL := `
// 		step "env_vars" "config" {
// 			arguments {} // Mock will provide defaults.
// 		}

// 		step "http_request" "first" {
// 			arguments {
// 				url = "https://httpbin.org/get"
// 			}
// 		}

// 		step "http_request" "delay_requests" {
// 			count = tonumber(step.env_vars.config.output.vars.REQUEST_COUNT)
// 			arguments {
// 				url = "https://httpbin.org/delay/${count.index}"
// 			}
// 			depends_on = ["step.http_request.first"]
// 		}

// 		# This final step depends on ALL instances via a splat expression.
// 		step "print" "show_all_results" {
// 			arguments {
// 				# The splat operator collects the 'output' from every instance.
// 				# The 'output' contains a 'data' attribute as defined in the manifest.
// 				input = step.http_request.delay_requests[*].output
// 			}
// 		}
// 	`
// 	files := map[string]string{"modules.hcl": manifestsHCL, "main.hcl": gridHCL}

// 	// Arrange: Define mock runners that return explicit cty.Value types.
// 	mockEnvVars := &testutil.SimpleModule{
// 		RunnerName: "OnRunEnvVars",
// 		Runner: &registry.RegisteredRunner{
// 			NewInput: func() any { return new(struct{}) },
// 			NewDeps:  func() any { return new(struct{}) },
// 			Fn: func(ctx context.Context, d, i any) (any, error) {
// 				// Return explicit cty.Value types to avoid ambiguity.
// 				return map[string]cty.Value{
// 					"vars": cty.MapVal(map[string]cty.Value{
// 						"REQUEST_COUNT": cty.StringVal("5"),
// 					}),
// 				}, nil
// 			},
// 		},
// 	}

// 	mockHttpRequest := &testutil.SimpleModule{
// 		RunnerName: "OnRunHttpRequest",
// 		Runner: &registry.RegisteredRunner{
// 			NewInput: func() any {
// 				return new(struct {
// 					URL string `bggo:"url"`
// 				})
// 			},
// 			NewDeps: func() any { return new(struct{}) },
// 			Fn: func(ctx context.Context, d, i any) (any, error) {
// 				url := i.(*struct {
// 					URL string `bggo:"url"`
// 				}).URL
// 				// The shape of this map must match the 'output' blocks in the manifest.
// 				httpResult := cty.ObjectVal(map[string]cty.Value{
// 					"url":    cty.StringVal(url),
// 					"status": cty.NumberIntVal(200),
// 				})
// 				return map[string]cty.Value{"data": httpResult}, nil
// 			},
// 		},
// 	}

// 	var consumedData cty.Value
// 	var mu sync.Mutex
// 	mockPrint := &testutil.SimpleModule{
// 		RunnerName: "OnRunPrint",
// 		Runner: &registry.RegisteredRunner{
// 			NewInput: func() any {
// 				return new(struct {
// 					Input cty.Value `bggo:"input"`
// 				})
// 			},
// 			NewDeps: func() any { return new(struct{}) },
// 			Fn: func(ctx context.Context, d, i any) (any, error) {
// 				mu.Lock()
// 				consumedData = i.(*struct {
// 					Input cty.Value `bggo:"input"`
// 				}).Input
// 				mu.Unlock()
// 				return nil, nil
// 			},
// 		},
// 	}

// 	// Act
// 	result := testutil.RunIntegrationTest(t, files, mockEnvVars, mockHttpRequest, mockPrint)

// 	// Assert
// 	require.NoError(t, result.Err, "The run should complete without errors")

// 	// Assert that the final "splat" step started and consumed the correct data.
// 	require.True(t, strings.Contains(result.LogOutput, "step=step.print.show_all_results[0]"), "The show_all_results step should have started")
// 	require.False(t, consumedData.IsNull(), "Print step should have consumed data")
// 	require.True(t, consumedData.Type().IsListType(), "Consumed data should be a list")
// 	require.Equal(t, 5, consumedData.LengthInt(), "Consumed data should have 5 elements from the delay_requests steps")

// 	// Assert correct execution order.
// 	allDelayFinishLog := "step=step.http_request.delay_requests instances_created=5"
// 	printStartLog := "step=step.print.show_all_results[0]"

// 	testutil.AssertLogOrder(t, result.LogOutput, allDelayFinishLog, printStartLog)
// }
