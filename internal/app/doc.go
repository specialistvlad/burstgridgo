// Package app contains the top-level application logic. It defines the main App
// struct, which acts as a high-level orchestrator for a run.
//
// The App's primary responsibility is to initiate an execution 'Session' by
// delegating to a session.Factory. This factory abstracts away the details of
// whether the run is local or distributed. Once a session is created, the App
// drives the execution by retrieving and running the session's Executor.
//
// This package is decoupled from any specific entrypoint like a CLI or server.
package app
