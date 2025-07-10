// Package main provides the GoKit CLI tool for managing Go web development tasks.
//
// The GoKit CLI is a command-line interface for common development tasks related to
// the GoKit library, particularly internationalization (i18n) management.
//
// Features:
//   - i18n file management (find missing keys, validate locales, extract keys)
//   - Extensible command structure for future tools
//   - Cross-platform compatibility
//
// Usage:
//
//	gokit <command> [options]
//
// Commands:
//
//	i18n    Manage internationalization files
//	help    Show help message
//
// Examples:
//
//	# Find missing translation keys
//	gokit i18n find-missing --source=en --target=es
//
//	# Validate locale files
//	gokit i18n validate --dir=./locales
//
//	# Extract keys from source code
//	gokit i18n extract --dir=./src --output=./locales
//
// Installation:
//
//	go install github.com/kdsmith18542/gokit/cmd/gokit@latest
package main

import (
	"fmt"
	"os"

	cli "github.com/kdsmith18542/gokit/cmd/gokit/i18n"
)

// main is the entry point for the GoKit CLI application.
// It parses command-line arguments and delegates to the appropriate command handler.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "i18n":
		cli.Run(args)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage displays the CLI usage information and available commands.
// This function is called when the user provides invalid arguments or requests help.
func printUsage() {
	fmt.Println("GoKit CLI - A toolkit for Go web development")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gokit <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  i18n    Manage internationalization files")
	fmt.Println("  help    Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gokit i18n find-missing --source=en --target=es")
	fmt.Println("  gokit i18n validate --dir=./locales")
	fmt.Println("  gokit i18n extract --dir=./src --output=./locales")
}
