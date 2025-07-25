# This grid tests the engine's resource management lifecycle (creation,
# usage by multiple steps, and destruction) using a hypothetical,
# purely in-memory "counter" resource.
#
# NOTE: This example uses 'local_counter' and 'counter_op' modules that
# would need to be implemented in Go for this grid to execute.

# 1. Define a stateful, local resource.
# The engine will create exactly one instance of this counter and share
# it with any steps that 'use' it. Its Go 'create' handler would simply
# return a new, thread-safe counter object initialized to zero.
resource "local_counter" "shared_tally" {
  # This hypothetical resource requires no configuration arguments.
}

# 2. First step: Increment the counter.
# This step 'uses' the shared resource. The engine injects the live
# counter object from "shared_tally" into this step's Go handler.
step "counter_op" "increment_first" {
  uses {
    counter = resource.local_counter.shared_tally
  }
  arguments {
    action = "increment"
  }
}

# 3. Second step: Increment the same counter again.
# This step also uses the *same* resource instance, demonstrating that
# state is maintained across steps. The explicit dependency ensures
# the increments happen in order.
step "counter_op" "increment_second" {
  uses {
    counter = resource.local_counter.shared_tally
  }
  arguments {
    action = "increment"
  }
  depends_on = [
    "counter_op.increment_first"
  ]
}


# 4. Final step: Get the final value from the counter.
# Its 'get' action would read the current value without changing it.
# The output of this step is what the test will assert against.
step "counter_op" "get_final_value" {
  uses {
    counter = resource.local_counter.shared_tally
  }
  arguments {
    action = "get"
  }
  depends_on = [
    "counter_op.increment_second"
  ]
}