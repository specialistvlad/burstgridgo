# ADR-005: Aligning HCL Manifest and Go Implementation Contracts
- **Status**: Implemented
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-28

---
## 1. Context

The project's architecture intends for HCL manifest files (e.g., `modules/http_request/manifest.hcl`) to be the **public contract** and single source of truth for a runner's API. The corresponding Go structs in the implementation (e.g., `modules/http_request/module.go`) are intended to be a private convenience for the developer.

An issue was discovered where the engine was not strictly enforcing this contract. It was possible to comment out all `input` and `output` blocks in a manifest, yet the runner would still function by decoding arguments directly into the Go struct based on `hcl` tags. This created a "two contracts" problem, where the implementation details (the Go struct) were leaking into the public API and the manifest was not being treated as the source of truth.

While immediate validation has been added to enforce the manifest's declarations at runtime, the root cause—the dual definition—still exists.

---
## 2. Decision

This ADR captures potential future solutions to resolve the dual-contract issue and create a true single source of truth for module definitions. After discussion, **Option 2 has been selected as the official path forward.**

### Option 1: HCL as the Single Source of Truth (Code Generation)

Under this model, the HCL manifest would be the only place a module's inputs and outputs are defined. We would introduce a build-time tool (e.g., using `go generate`) that would parse the manifest and automatically generate the corresponding Go `Input` struct.

* **Decision**: This option was **rejected** due to the added complexity of a code generation step in the build process, which could negatively impact the developer experience for new contributors.

### Option 2: Strict Manifest/Implementation Parity Check (Startup-Time Reflection)

This model would keep both definitions (HCL and Go) but would add a strict, one-time validation check when the application starts. The engine would use reflection to inspect the Go `Input` struct for a registered runner and compare it field-for-field against the declared `input` blocks in its HCL manifest. The application would panic and refuse to start if there were any discrepancies (e.g., a field in the Go struct without a matching `input` block, or vice-versa).

* **Decision**: This option was **accepted** as the preferred solution. It provides the necessary safety and contract enforcement without adding complexity to the build process. It is a pragmatic compromise that fails fast during development if the two contracts are out of sync.

---
## 3. Consequences

The immediate fix of adding runtime validation has already improved system robustness. The accepted plan is to now implement the **Strict Manifest/Implementation Parity Check** at application startup.

This will provide the ultimate guarantee that the HCL manifest is the single source of truth for all modules. It eliminates the "two contracts" problem in practice, improves maintainability, and ensures that module authors and users can rely on the manifest as the definitive public API. The implementation of this startup check is now on the project roadmap.