package main

import (
	"log"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	_ "github.com/vk/burstgridgo/modules/env_vars"
	_ "github.com/vk/burstgridgo/modules/http_request"
	_ "github.com/vk/burstgridgo/modules/print"
)

func main() {
	// 1. Parse all CLI arguments and flags.
	cliOpts, err := config.Parse()
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}

	// 2. Resolve the grid path to a list of HCL files.
	hclFiles, err := engine.ResolveGridPath(cliOpts.GridPath)
	if err != nil {
		log.Fatalf("Error resolving grid path '%s': %v", cliOpts.GridPath, err)
	}
	if len(hclFiles) == 0 {
		log.Fatalf("No .hcl files found in '%s'", cliOpts.GridPath)
	}

	// Log the discovered files for clarity.
	log.Printf("Found %d HCL file(s) to process from '%s':", len(hclFiles), cliOpts.GridPath)
	for _, file := range hclFiles {
		log.Printf("  • %s", file)
	}

	// 3. Parse all files to get a flat list of modules.
	var allModules []*engine.Module
	for _, file := range hclFiles {
		config, err := engine.DecodeHCLFile(file)
		if err != nil {
			log.Printf("❗️ Error decoding %s: %v", file, err)
			continue
		}
		allModules = append(allModules, config.Modules...)
	}

	// 4. Build the dependency graph.
	graph, err := dag.NewGraph(allModules)
	if err != nil {
		log.Fatalf("❗️ Error building dependency graph: %v", err)
	}

	// 5. Create an executor and run the graph.
	log.Println("🚀 Starting concurrent execution...")
	executor := dag.NewExecutor(graph)
	executor.Run()

	log.Println("🏁 Execution finished.")
}
