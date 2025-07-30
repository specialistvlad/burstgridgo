# Test Suite

This directory contains the integration and system-level tests for the `burstgridgo` application.

## Test Philosophy

The tests in this directory are designed to be **deterministic** and **isolated**. They validate the end-to-end behavior of the application by running the core engine in-process with mocked handlers for external interactions. This approach provides high confidence in the application's correctness without the flakiness or overhead of external network dependencies.

The structure and test cases are based on the plan outlined in [ADR-006: Comprehensive System Integration Testing](../../docs/features/ADR-006-system-integration-testing.md).

## Directory Structure

-   `/system`: Contains end-to-end system tests that exercise the application from the CLI layer down through the DAG executor.
    -   `helpers_test.go`: Contains shared helper functions for setting up test environments.
    -   `*_test.go`: Test files are organized by feature category, as defined in ADR-006.

## Running Tests

These tests use Go's build tags to allow for granular test execution.

### Run Only System Tests

To run all system tests (those in the `test/system` directory):

```sh
make test
```