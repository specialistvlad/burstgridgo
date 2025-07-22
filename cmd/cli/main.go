package main

import (
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/vk/burstgridgo/internal/healthcheck"
	_ "github.com/vk/burstgridgo/modules/env_vars"
	_ "github.com/vk/burstgridgo/modules/help"
	_ "github.com/vk/burstgridgo/modules/http_request"
	_ "github.com/vk/burstgridgo/modules/print"
	_ "github.com/vk/burstgridgo/modules/socketio"
)

func main() {
	// 1. Parse all CLI arguments and flags.
	cliOpts, err := config.Parse()
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}

	// Start the health check server if the port is configured.
	if cliOpts.HealthcheckPort > 0 {
		go healthcheck.StartServer(cliOpts.HealthcheckPort)
	}

	var allModules []*engine.Module

	// 2. If a grid path is provided, find and parse the HCL files.
	if cliOpts.GridPath != "" {
		hclFiles, err := engine.ResolveGridPath(cliOpts.GridPath)
		if err != nil {
			log.Fatalf("Error resolving grid path '%s': %v", cliOpts.GridPath, err)
		}

		if len(hclFiles) > 0 {
			log.Printf("Found %d HCL file(s) to process from '%s':", len(hclFiles), cliOpts.GridPath)
			for _, file := range hclFiles {
				log.Printf("  • %s", file)
			}

			// Parse all files to get a flat list of modules.
			for _, file := range hclFiles {
				config, err := engine.DecodeHCLFile(file)
				if err != nil {
					log.Printf("❗️ Error decoding %s: %v", file, err)
					continue
				}
				allModules = append(allModules, config.Modules...)
			}
		}
	}

	// 3. If no modules were loaded, inject the help module.
	if len(allModules) == 0 {
		if cliOpts.GridPath != "" {
			log.Printf("No modules found in '%s'. Displaying help.", cliOpts.GridPath)
		}
		// Create a single module to run the help runner.
		allModules = []*engine.Module{
			{
				Name:   "show_help",
				Runner: "help",
				Body:   hcl.EmptyBody(), // Corrected: Call the function with ()
			},
		}
	}

	// 4. Build the dependency graph.
	graph, err := dag.NewGraph(allModules)
	if err != nil {
		log.Fatalf("❗️ Error building dependency graph: %v", err)
	}

	// 5. Create an executor and run the graph.
	if len(graph.Nodes) > 0 {
		log.Println("🚀 Starting concurrent execution...")
		executor := dag.NewExecutor(graph)
		executor.Run()
		log.Println("🏁 Execution finished.")
	}
}
