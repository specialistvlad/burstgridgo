package integration_tests

import (
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStepParsing_Attributes verifies the parsing of all simple, top-level attributes in a step.
func TestStepParsing_Attributes(t *testing.T) {
	cases := []testutil.StepTestCase{
		{Name: "attribute enabled", HCL: `enabled = false`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Enabled) }},
		{Name: "attribute description", HCL: `description = "a test step"`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Description) }},
		{Name: "attribute tags", HCL: `tags = ["api", "critical"]`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Tags) }},
		{Name: "attribute scope", HCL: `scope = "workspace"`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Scope) }},
		{Name: "attribute uses", HCL: `uses = [resource.db.primary]`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Uses) }},
		{Name: "attribute priority", HCL: `priority = 100`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Priority) }},
		{Name: "attribute delay_before", HCL: `delay_before = "10s"`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.DelayBefore) }},
		{Name: "attribute delay_after", HCL: `delay_after = "500ms"`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.DelayAfter) }},
		{Name: "attribute continue_on_failure", HCL: `continue_on_failure = true`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.ContinueOnFailure) }},
		{Name: "attribute idempotency_key", HCL: `idempotency_key = var.transaction_id`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.IdempotencyKey) }},
		{Name: "attribute sensitive", HCL: `sensitive = true`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Sensitive) }},
		{Name: "attribute env", HCL: `env = { API_KEY = var.api_key }`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Env) }},
	}
	testutil.RunStepParsingTests(t, cases)
}

// TestStepParsing_Blocks verifies the parsing of all nested blocks within a step.
func TestStepParsing_Blocks(t *testing.T) {
	cases := []testutil.StepTestCase{
		// Arguments Block
		{Name: "block arguments", HCL: `
			arguments {
				message = "hello"
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Arguments)
			require.Contains(t, s.Arguments, "message")
		}},
		{Name: "block arguments duplicate", HCL: `
			arguments {}
			arguments {}`, ExpectErr: true, ErrContains: `Duplicate "arguments" block`},

		// Execution Control Blocks
		{Name: "block timeouts", HCL: `
			timeouts {
				execution = "5m"
				start     = "1m"
				queue     = "30s"
				deadline  = "2025-08-16T18:00:00Z"
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Timeouts)
			require.NotNil(t, s.Timeouts.Execution)
			require.NotNil(t, s.Timeouts.Start)
			require.NotNil(t, s.Timeouts.Queue)
			require.NotNil(t, s.Timeouts.Deadline)
		}},

		// Concurrency Blocks
		{Name: "block concurrency", HCL: `
			concurrency {
				limit   = 50
				per_key = var.user_id
				order   = "ordered"
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Concurrency)
			require.NotNil(t, s.Concurrency.Limit)
			require.NotNil(t, s.Concurrency.PerKey)
			require.NotNil(t, s.Concurrency.Order)
		}},
		{Name: "block rate_limit", HCL: `
			rate_limit {
				limit = 100
				per   = "1m"
				burst = 10
				key   = var.tenant_id
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.RateLimit)
			require.NotNil(t, s.RateLimit.Limit)
			require.NotNil(t, s.RateLimit.Per)
			require.NotNil(t, s.RateLimit.Burst)
			require.NotNil(t, s.RateLimit.Key)
		}},

		// Error Handling Blocks
		{Name: "block on_error", HCL: `
			on_error {
				action   = "fallback"
				fallback = step.http.cleanup
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.OnError)
			require.NotNil(t, s.OnError.Action)
			require.NotNil(t, s.OnError.Fallback)
		}},
		{Name: "block retry", HCL: `
			retry {
				attempts     = 5
				max_duration = "10m"
				retry_on     = ["timeout"]
				abort_on     = ["404"]
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Retry)
			require.NotNil(t, s.Retry.Attempts)
			require.NotNil(t, s.Retry.MaxDuration)
			require.NotNil(t, s.Retry.RetryOn)
			require.NotNil(t, s.Retry.AbortOn)
		}},
		{Name: "block retry with backoff", HCL: `
			retry {
				backoff {
					strategy = "exponential"
					initial  = "1s"
					factor   = 2
					max      = "1m"
					jitter   = "full"
				}
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Retry.Backoff)
			require.NotNil(t, s.Retry.Backoff.Strategy)
			require.NotNil(t, s.Retry.Backoff.Initial)
			require.NotNil(t, s.Retry.Backoff.Factor)
			require.NotNil(t, s.Retry.Backoff.Max)
			require.NotNil(t, s.Retry.Backoff.Jitter)
		}},

		// State Blocks
		{Name: "block cache", HCL: `
			cache {
				enabled = true
				key     = "k"
				ttl     = "1h"
				scope   = "global"
				restore = "on_retry"
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Cache)
			require.NotNil(t, s.Cache.Enabled)
			require.NotNil(t, s.Cache.Key)
			require.NotNil(t, s.Cache.TTL)
			require.NotNil(t, s.Cache.Scope)
			require.NotNil(t, s.Cache.Restore)
		}},
		{Name: "block dedupe", HCL: `
			dedupe {
				key    = var.key
				action = "cancel_previous"
				scope  = "workspace"
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Dedupe)
			require.NotNil(t, s.Dedupe.Key)
			require.NotNil(t, s.Dedupe.Action)
			require.NotNil(t, s.Dedupe.Scope)
		}},

		// Observability Blocks
		{Name: "block tracing", HCL: `
			tracing {
				attributes  = {k="v"}
				sample_rate = 0.5
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Tracing)
			require.NotNil(t, s.Tracing.Attributes)
			require.NotNil(t, s.Tracing.SampleRate)
		}},
		{Name: "block metrics", HCL: `
			metrics {
				emit = ["latency"]
			}`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Metrics); require.NotNil(t, s.Metrics.Emit) }},

		// Placement Block
		{Name: "block placement", HCL: `
			placement {
				labels      = {r="w"}
				constraints = ["a"]
				shard_by    = var.id
			}`, Validate: func(t *testing.T, s *model.Step) {
			require.NotNil(t, s.Placement)
			require.NotNil(t, s.Placement.Labels)
			require.NotNil(t, s.Placement.Constraints)
			require.NotNil(t, s.Placement.ShardBy)
		}},
	}
	testutil.RunStepParsingTests(t, cases)
}

// TestStepParsing_Looping verifies parsing and validation of 'count' and 'for_each'.
func TestStepParsing_Looping(t *testing.T) {
	cases := []testutil.StepTestCase{
		{Name: "looping count valid integer", HCL: `count = 5`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Count) }},
		{Name: "looping count valid variable", HCL: `count = var.num`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.Count) }},
		{Name: "looping count invalid float", HCL: `count = 5.5`, ExpectErr: true, ErrContains: "must be a whole number"},
		{Name: "looping count invalid string", HCL: `count = "five"`, ExpectErr: true, ErrContains: "Invalid count value"},
		{Name: "looping for_each valid list", HCL: `for_each = ["a", "b"]`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.ForEach) }},
		{Name: "looping for_each valid variable", HCL: `for_each = var.list`, Validate: func(t *testing.T, s *model.Step) { require.NotNil(t, s.ForEach) }},
		{Name: "looping for_each invalid literal type", HCL: `for_each = 123`, ExpectErr: true, ErrContains: "Invalid for_each value"},
		{Name: "looping for_each invalid mixed list", HCL: `for_each = ["a", 1]`, ExpectErr: true, ErrContains: "all elements must be strings"},
		{Name: "looping conflict count and for_each", HCL: `
			count    = 2
			for_each = []
			`, ExpectErr: true, ErrContains: "Conflicting looping attributes"},
	}
	testutil.RunStepParsingTests(t, cases)
}

// TestStepParsing_Dependencies verifies parsing and validation of 'depends_on'.
func TestStepParsing_Dependencies(t *testing.T) {
	cases := []testutil.StepTestCase{
		{Name: "depends_on valid list", HCL: `depends_on = [step.print.first]`, Validate: func(t *testing.T, s *model.Step) { assert.Len(t, s.Expressions.References(), 1) }},
		{Name: "depends_on valid empty list", HCL: `depends_on = []`, Validate: func(t *testing.T, s *model.Step) { assert.Empty(t, s.Expressions.References()) }},
		{Name: "depends_on invalid scalar", HCL: `depends_on = step.x.y`, ExpectErr: true, ErrContains: "must be a list of step references"},
	}
	testutil.RunStepParsingTests(t, cases)
}

// TestStepParsing_Structure verifies high-level structural rules of a step block.
func TestStepParsing_Structure(t *testing.T) {
	cases := []testutil.StepTestCase{
		{Name: "structure unknown attribute", HCL: `non_existent_attr = true`, ExpectErr: true, ErrContains: "Unsupported argument"},
		{Name: "structure unknown block", HCL: `
			unexpected_block {}
			`, ExpectErr: true, ErrContains: "Unsupported block type"},
	}
	testutil.RunStepParsingTests(t, cases)
}
