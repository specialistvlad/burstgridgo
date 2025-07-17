# BurstGridGo: Foundational Design Summary

This document captures key design decisions, patterns, and concepts discussed for the BurstGridGo project — a declarative, modular, and concurrency-first load testing tool written in Go using HCL.

---

## ✅ Project Vision

- Simulate real-world workflows across diverse protocols (HTTP, Socket.IO, etc.)
- Provide a developer-first experience with minimal boilerplate
- Use **HCL** (HashiCorp Configuration Language) as the declarative config format
- Inspired by Terraform and Go concurrency idioms
- Designed to be reusable, extensible, and composable

---

## 🧾 Configuration Language: HCL

- BurstGridGo will use **HCL** (`hcl/v2`, `gohcl`, `cty`) for configuration
- Users define reusable units (`module`) with optional input/output and flow control
- Example syntax:

```hcl
module "upload_file" {
  method = "POST"
  url    = "/api/upload"
  file   = "./demo.bin"
}
```

- Inspired by Terraform: structured blocks, interpolation, modular composition
- Tooling to reuse: `hcl/v2`, `gohcl`, `hclwrite`, `cty`, `terraform-config-inspect`

---

## ⚙️ Core Concepts

### 1. **Module**

- The smallest executable unit (e.g. an HTTP request, socket action, custom runner)
- Can be backed by built-in logic or user-defined Go code
- Can live in the same file or be split into multiple files/folders
- Supports config + optional inline code per module

### 2. **Execution Model**

- **Concurrency by default**
- `depends_on = ["other_module"]` d efines explicit dependencies
- Execution graph (DAG) is built from dependencies and output references

### 3. **Lifecycle**

- Determined **automatically** by dependency graph:
  - If a module has dependents → stays alive until all finish
  - If a module has no dependents → runs and exits
- This allows persistent modules like `socketio` without requiring manual lifecycle config

### 4. **Outputs and Interpolation** *(planned)*

- Modules can expose `outputs`
- Downstream modules can use `${module.upload.output_id}` style references
- Creates **implicit dependencies** if interpolation is detected

---

## 🔧 Example Minimal Config (Flat Execution)

```hcl
module "health_check" {
  runner = "http"

  method = "GET"
  url    = "http://host.docker.internal:15060/health-check"

  expect = {
    status = 200
  }
}
```

---

## 🔄 Example DAG-Based Config (Flow Execution)

```hcl
module "connect_socket" {
  socketio = {
    url = "wss://example.com/ws"
  }
}

module "request_upload" {
  method     = "POST"
  url        = "/api/request-upload"
  depends_on = ["connect_socket"]
}

module "upload_file" {
  method     = "PUT"
  url        = "https://s3.example.com/upload"
  file       = "./demo.bin"
  depends_on = ["request_upload"]
}

module "wait_updates" {
  socketio = {
    event = "progress"
  }
  depends_on = ["connect_socket", "upload_file"]
}

module "generate_stats" {
  runner     = "stats"
  depends_on = ["wait_updates"]
}
```
