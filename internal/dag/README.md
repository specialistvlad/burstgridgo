# DAG Package README

The `dag` package is the **Execution Layer** of the application. Its sole responsibility is to take a `GridConfig` blueprint from the `engine`, build a Directed Acyclic Graph (DAG) of nodes, and execute the nodes concurrently according to their dependencies.

## Core Responsibilities

### 1. Graph Construction
The `NewGraph` constructor takes a `GridConfig` and builds the full execution graph. This is a multi-pass process:
1.  **Node Creation**: A `Node` is created for every `step` and `resource` in the configuration and stored in the `Graph.Nodes` map, keyed by its unique ID (e.g., `step.http_request.my_step`).
2.  **Node Linking**: The engine iterates through all nodes and links them based on dependencies.
    -   **Implicit Deps**: It inspects HCL bodies for variable traversals (e.g., `step.A.output`) to create dependency links.
    -   **Explicit Deps**: It resolves identifiers from `depends_on = [...]` blocks and the `uses {}` block to create links.
3.  **Counter Initialization**: It initializes the `depCount` (for scheduling) and `descendantCount` (for resource cleanup) atomic counters on each node.
4.  **Cycle Detection**: It performs a DFS traversal to ensure the graph is a valid DAG and contains no circular dependencies.

### 2. The Executor
The `Executor` is the primary struct responsible for running the graph.
-   **Worker Pool**: The `Executor.Run()` method starts a pool of Go `worker` goroutines.
-   **Ready Channel**: A `readyChan` is used to feed the workers. Nodes with a `depCount` of zero are placed on this channel at the start.
-   **Node Execution**: When a worker picks up a `Node`, it delegates the actual execution to helper functions based on the node's type: `executeStepNode` or `executeResourceNode`. These helpers are responsible for building the HCL evaluation context, decoding arguments, building the `Deps` struct for injection, and calling the appropriate Go handler from the `engine` registries.
-   **Unlocking Dependents**: Upon successful completion of a node, the executor decrements the `depCount` of all its dependents. If a dependent's `depCount` reaches zero, it is placed on the `readyChan`.

### 3. Resource Lifecycle Management
The `dag` package is responsible for the `Create` and `Destroy` lifecycle of resources.
-   **Central Store**: A `sync.Map` (`resourceInstances`) holds the live, stateful Go objects returned by a resource's `Create` handler.
-   **Efficient Cleanup**: When a step completes, it decrements the `descendantCount` of all the resources it `uses`. If a resource's counter reaches zero, `destroyResource` is called on it in a new goroutine.
-   **Guaranteed Cleanup**: When a resource is successfully created, its `Destroy` function is pushed onto a LIFO `cleanupStack`. A `defer e.executeCleanupStack()` in the main `Run()` method ensures all created resources are cleaned up, even if the run fails.