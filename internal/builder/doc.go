/*
Package builder is responsible for the architectural construction of the execution
graph. It acts as the bridge between the static configuration model (defined in
the 'config' package) and the dynamic execution engine (the 'executor' package).

The primary artifact produced by this package is a validated, ready-to-run *Graph.

The graph construction is a multi-phase process:

 1. Node Creation: The builder iterates through the configuration's steps and
    resources, creating a corresponding *node.Node for each one. This phase populates the
    graph with its vertices but does not yet establish their relationships.

 2. Dependency Linking: The builder analyzes the `depends_on` attributes (explicit
    dependencies) and variable references within expressions (implicit dependencies)
    for each Node. It uses this analysis to create directed edges in the graph,
    forming a complete dependency topology. This work is delegated to the generic,
    thread-safe `dag` package.

 3. Validation and Initialization: Once the graph topology is complete, the builder
    performs two final actions:
    a. It invokes the DAG's cycle detection algorithm to ensure the graph is
    acyclic and thus executable.
    b. It initializes scheduler-critical counters on each Node (e.g., the number
    of unmet dependencies), preparing the graph for the executor.

Upon successful completion, the builder hands off the fully constructed and
validated *Graph to the executor, which is then responsible for orchestrating
the execution of the Nodes.
*/
package builder
