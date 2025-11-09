package main

import (
	"fmt"
	"log"
	"os"

	"dev-mcp/internal/config"
	"dev-mcp/internal/mcp/server"
)

func main() {
	// Check for debug mode (for future use)
	for _, arg := range os.Args {
		if arg == "--debug" || arg == "-d" {
			fmt.Println("Debug mode enabled")
			break
		}
	}

	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	server.NewMCPServer(cfg)

	fmt.Println("\nTo start as MCP server, run: go run cmd/main.go mcp")
}
