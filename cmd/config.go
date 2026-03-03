package cmd

import (
	"fmt"

	"github.com/alexmx/skillman/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		fmt.Printf("Config file:  %s\n", config.ConfigPath())
		fmt.Printf("Store path:   %s\n", cfg.ResolvedStorePath())
		fmt.Println()
		fmt.Println("Agents:")
		for name, agent := range cfg.Agents {
			status := "disabled"
			if agent.Enabled {
				status = "enabled"
			}
			fmt.Printf("  %-10s %s (path: %s)\n", name, status, agent.SkillPath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
