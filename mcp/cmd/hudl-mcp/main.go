package main

import (
	"fmt"
	"os"

	"github.com/Huddle01/get-hudl/mcp/internal/server"
	"github.com/Huddle01/get-hudl/mcp/internal/tools"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("hudl-mcp %s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Fprintf(os.Stderr, "hudl-mcp %s — Huddle01 Cloud MCP server\n\n", version)
			fmt.Fprintf(os.Stderr, "Usage: hudl-mcp\n\n")
			fmt.Fprintf(os.Stderr, "Runs a Model Context Protocol (MCP) server over stdio.\n")
			fmt.Fprintf(os.Stderr, "Configure your MCP client to launch this binary.\n\n")
			fmt.Fprintf(os.Stderr, "Authentication:\n")
			fmt.Fprintf(os.Stderr, "  Uses ~/.hudl/config.toml or HUDL_API_KEY env var.\n")
			fmt.Fprintf(os.Stderr, "  Run `hudl login --token <key>` first, or use the hudl_login tool.\n\n")
			fmt.Fprintf(os.Stderr, "Flags:\n")
			fmt.Fprintf(os.Stderr, "  --version    Print version and exit\n")
			fmt.Fprintf(os.Stderr, "  --help       Print this help and exit\n")
			return
		}
	}

	srv := server.New("hudl-mcp", version)
	tools.RegisterAll(srv)

	if err := srv.Run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "hudl-mcp: %v\n", err)
		os.Exit(1)
	}
}
