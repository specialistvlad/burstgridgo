// Package executor is responsible for running a pre-built execution graph (dag.Graph).
// It manages a worker pool, executes nodes concurrently according to their
// dependencies, handles resource lifecycle (creation and cleanup), and manages
// the overall execution state.
package old_executor
