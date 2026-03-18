package cmd

import (
	"fmt"
	"os"

	"github.com/alexmx/skillman/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "skillman",
	Short:   "A package manager for Agent Skills",
	Long:    "Skillman manages Agent Skills — install from GitHub or local paths into your workspace for any supported AI coding agent.",
	Version: version.Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}
