// Package dag is responsible for building a validated execution graph. It defines
// the core data structures for the graph (Node, Graph) and contains the logic
// for translating a GridConfig into a Directed Acyclic Graph (DAG),
// resolving dependencies, and detecting cycles.
//
// This package does not execute the graph; that is the responsibility of the
// 'executor' package.
package dag
