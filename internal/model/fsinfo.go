// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// // This file defines the FSInfo struct, which stores file system metadata.
//
// Why store the file path?
//
// The file path is a critical piece of metadata that connects a parsed in-memory
// object (like a Step or Runner) back to its physical source on disk. This
// information is foundational for several key architectural features:
//
//  1. **Error Reporting**: It is invaluable for providing clear, actionable error
//     messages. The system can report not just *what* is wrong, but also exactly
//     *in which file* the problematic definition is located.
//
//  2. **Module Construction**: In a file-based module system, the directory
//     containing a definition file often defines the boundary of a module. The
//     file path is the primary input for determining which module an object
//     belongs to.
//
//  3. **Scope Resolution**: The resolution of scopes (e.g., `local`, `module`,
//     `workspace`) is context-dependent. The file path provides the necessary
//     context to determine which other definitions are "visible" to the current
//     one, enabling the correct resolution of dependencies within a given scope.
package model

type FSInfo struct {
	FilePath string
}

func NewFSInfo(filePath string) *FSInfo {
	return &FSInfo{
		FilePath: filePath,
	}
}
