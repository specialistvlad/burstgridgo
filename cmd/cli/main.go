package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type HTTPExpect struct {
	Status int `hcl:"status"`
}

type Module struct {
	Name   string      `hcl:"name,label"`
	Runner string      `hcl:"runner"`
	Method string      `hcl:"method,optional"`
	URL    string      `hcl:"url,optional"`
	Expect *HTTPExpect `hcl:"expect,block"`
}

func main() {
	folder := flag.String("folder", "examples/http_request", "folder path to scan for .hcl files")
	flag.Parse()

	parser := hclparse.NewParser()

	err := filepath.Walk(*folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".hcl" {
			fmt.Println("Parsing file:", path)
			file, diag := parser.ParseHCLFile(path)
			if diag.HasErrors() {
				fmt.Println("  Failed to parse:", diag.Error())
			} else {
				fmt.Println("  Parsed successfully.")

				body := file.Body
				var content struct {
					Modules []Module `hcl:"module,block"`
				}
				diags := gohcl.DecodeBody(body, nil, &content)
				if diags.HasErrors() {
					fmt.Println("  Failed to decode modules:", diags.Error())
				} else {
					for _, m := range content.Modules {
						fmt.Println("  Found module:", m.Runner)
						// engine.ExecuteModule(m)
					}

					// Output the whole config in JSON format
					// Marshal the content struct to JSON
					jsonData, err := json.MarshalIndent(content, "", "  ")
					if err != nil {
						fmt.Println("  Failed to marshal to JSON:", err)
					} else {
						fmt.Println("  JSON Output:")
						fmt.Println(string(jsonData))
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error scanning folder:", err)
	}
}
