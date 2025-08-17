package integration_tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// instancingTestModule provides mock runners for testing instancing features.
type instancingTestModule struct {
	// consumedValue will store the value received by the 'consumer' step.
	// We expect it to be the output of a specific 'source' instance.
	consumedValue cty.Value
}

func (m *instancingTestModule) Register(r *registry.Registry) {
	// A runner that receives its index via arguments and outputs it.
	type sourceInput struct {
		Index int `bggo:"idx"`
	}
	// Define a concrete output struct for the source runner.
	type sourceOutput struct {
		InstanceIndex int  `cty:"instance_index"`
		IsCorrect     bool `cty:"is_correct"`
	}
	r.RegisterRunner("OnRunSource", &registry.RegisteredRunner{
		NewInput:  func() any { return new(sourceInput) },
		InputType: reflect.TypeOf(sourceInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (any, error) {
			in := input.(*sourceInput)
			// Return a concrete struct, not a generic map.
			return &sourceOutput{
				InstanceIndex: in.Index,
				IsCorrect:     in.Index == 1,
			}, nil
		},
	})

	// A runner that consumes the output of a source step.
	type consumerInput struct {
		InputValue cty.Value `bggo:"input_val"`
	}
	r.RegisterRunner("OnRunConsumer", &registry.RegisteredRunner{
		NewInput:  func() any { return new(consumerInput) },
		InputType: reflect.TypeOf(consumerInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (any, error) {
			in := input.(*consumerInput)
			m.consumedValue = in.InputValue
			return nil, nil
		},
	})
}

func TestCoreExecution_Instancing_SuccessfulDependency(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	sourceManifest := `
        runner "source" {
          lifecycle { on_run = "OnRunSource" }
          input "idx" {
            type = number
          }
          output "output" {
            type = object({
              instance_index = number,
              is_correct     = bool
            })
          }
        }
    `
	consumerManifest := `
        runner "consumer" {
          lifecycle { on_run = "OnRunConsumer" }
          input "input_val" {
            type = any
          }
        }
    `
	gridHCL := `
        step "source" "many" {
          count = 3 // Creates instances [0], [1], and [2]

          // Pass the instance index to the runner so it can output it.
          arguments {
            idx = count.index
          }
        }

        step "consumer" "one" {
          // Implicitly depend on the output of instance [1].
          arguments {
            input_val = step.source.many[1].output
          }

          // Explicitly depend on instance [2] as well for ordering.
          depends_on = [
            "source.many[2]"
          ]
        }
    `
	files := map[string]string{
		"modules/source.hcl":   sourceManifest,
		"modules/consumer.hcl": consumerManifest,
		"main.hcl":             gridHCL,
	}
	mockModule := &instancingTestModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "test run should succeed")
	require.NotNil(t, mockModule.consumedValue, "consumer should have received a value")
	require.True(t, mockModule.consumedValue.IsKnown() && !mockModule.consumedValue.IsNull(), "consumed value must be known and not null")
	require.True(t, mockModule.consumedValue.Type().IsObjectType(), "consumed value should be an object type")

	// Verify that the consumer received the output from the correct instance ([1]).
	valMap := mockModule.consumedValue.AsValueMap()
	indexVal, ok := valMap["instance_index"]
	require.True(t, ok, "output should have an 'instance_index' attribute")

	isCorrectVal, ok := valMap["is_correct"]
	require.True(t, ok, "output should have an 'is_correct' attribute")

	require.Equal(t, cty.NumberIntVal(1), indexVal, "should have received output from index 1")
	require.Equal(t, cty.True, isCorrectVal, "the 'is_correct' flag should be true")
}
