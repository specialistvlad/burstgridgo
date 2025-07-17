package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/vk/burstgridgo/internal/engine"
	// Blank import for its side-effect: registering the runner in its init() function.
	_ "github.com/vk/burstgridgo/modules/http_request"
)

func main() {
	folder := flag.String("folder", "examples", "folder path to scan for .hcl files")
	flag.Parse()

	hclFiles, err := engine.FindHCLFiles(*folder)
	if err != nil {
		log.Fatalf("Error scanning folder '%s': %v", *folder, err)
	}

	for _, file := range hclFiles {
		fmt.Println("Processing file:", file)

		genericConfig, err := engine.DecodeHCLFile(file)
		if err != nil {
			log.Printf("  ❗️ Error decoding generic config from %s: %v", file, err)
			continue
		}

		// Dispatch modules using the registry.
		for _, module := range genericConfig.Modules {
			fmt.Printf("  ▶️ Found module '%s' with runner '%s'\n", module.Name, module.Runner)

			// Look up the runner in the registry.
			if runner, ok := engine.Registry[module.Runner]; ok {
				// If found, execute it.
				if err := runner.Run(*module); err != nil {
					log.Printf("    ❗️ Error executing module '%s': %v", module.Name, err)
				}
			} else {
				log.Printf("    ❓ Unknown runner type '%s' for module '%s'", module.Runner, module.Name)
			}
		}
	}
}
