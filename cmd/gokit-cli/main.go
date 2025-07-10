package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gokit-cli",
		Short: "GoKit CLI - Unified toolkit for Go web development",
	}

	// Add i18n subcommand
	rootCmd.AddCommand(I18nCmd())
	// Add upload subcommand
	rootCmd.AddCommand(UploadCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
