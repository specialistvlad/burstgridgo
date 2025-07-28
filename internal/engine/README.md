# Engine Package README

The `engine` package is the **Configuration and Loading Layer** of the application. It is responsible for discovering, parsing, and loading all HCL-based definitions and preparing the final `GridConfig` blueprint that the `dag` package executes.

## Core Responsibilities

### 1. Module Discovery
The `DiscoverModules` function is the entrypoint for finding all `runner` and `asset` definitions. It scans the `modules/` directory for HCL manifest files, decodes them, and populates the global `DefinitionRegistry` and `AssetDefinitionRegistry`.

### 2. Grid Loading
The `LoadGridConfig` function is the primary public function for processing a user's request. It encapsulates the entire loading process:
-   **Path Resolution**: Takes a path to a file or directory and resolves it to a list of `.hcl` files using `ResolveGridPath`.
-   **Parsing & Decoding**: Iterates through each file, calling `DecodeGridFile` to parse it into an intermediate `GridConfig` struct.
-   **Merging**: Appends all decoded `Steps` and `Resources` from multiple files into a single, unified `GridConfig`.

### 3. The Registries
This package defines all the global, singleton registries that connect HCL definitions to their Go implementations. These are populated at program startup.
-   `DefinitionRegistry`: A map of `[runner_type] -> *RunnerDefinition`. Stores the parsed HCL manifests for all runners.
-   `AssetDefinitionRegistry`: A map of `[asset_type] -> *AssetDefinition`. Stores the parsed HCL manifests for all assets.
-   `HandlerRegistry`: A map of `[handler_name] -> *RegisteredHandler`. Stores the registered Go functions for runner lifecycles.
-   `AssetHandlerRegistry`: A map of `[handler_name] -> *RegisteredAssetHandler`. Stores the registered Go functions for asset lifecycles.
-   `AssetInterfaceRegistry`: A map of `[asset_type] -> reflect.Type`. Stores the Go interface associated with an asset type, enabling type-safe dependency injection.

### 4. HCL Schemas
All of the Go structs that represent HCL blocks (`runner`, `step`, `input`, `resource`, etc.) are defined in `schema.go` and `engine.go`. These structs use `hcl` tags to enable decoding by the `gohcl` library.