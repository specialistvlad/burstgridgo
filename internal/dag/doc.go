// Package dag is the "Execution Layer" of the application. It is responsible
// for taking a GridConfig blueprint from the engine, building a Directed
// Acyclic Graph (DAG) of nodes, and executing the nodes concurrently according
// to their dependencies.
//
// For a detailed architectural overview, see the README.md file in this directory.
package dag
