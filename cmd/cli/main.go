package main

import (
	"flag"
	"log"

	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	_ "github.com/vk/burstgridgo/modules/http_request"
)

func main() {
	folder := flag.String("folder", "examples", "folder path to scan for .hcl files")
	flag.Parse()

	// 1. Find all HCL files.
	hclFiles, err := engine.FindHCLFiles(*folder)
	if err != nil {
		log.Fatalf("Error scanning folder '%s': %v", *folder, err)
	}

	// 2. Parse all files to get a flat list of modules.
	var allModules []*engine.Module
	for _, file := range hclFiles {
		config, err := engine.DecodeHCLFile(file)
		if err != nil {
			log.Printf("❗️ Error decoding %s: %v", file, err)
			continue
		}
		allModules = append(allModules, config.Modules...)
	}

	// 3. Build the dependency graph.
	graph, err := dag.NewGraph(allModules)
	if err != nil {
		log.Fatalf("❗️ Error building dependency graph: %v", err)
	}

	log.Println("🚀 Starting concurrent execution...")
	// 4. Create an executor and run the graph.
	executor := dag.NewExecutor(graph)
	executor.Run()

	log.Println("🏁 Execution finished.")
}
