package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/alexmx/skillman/internal/store"
	"github.com/alexmx/skillman/internal/tui"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:     "remove <skill-name>",
	Short:   "Remove a skill from the store",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		s := store.New(cfg)
		reg, err := registry.Load(cfg)
		if err != nil {
			return err
		}

		entry := reg.Find(name)
		if entry == nil {
			return fmt.Errorf("skill %q is not installed", name)
		}

		if !removeForce {
			yes, err := tui.Confirm(fmt.Sprintf("Remove %q from the store?", name))
			if err != nil {
				return err
			}
			if !yes {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Remove from store
		storePath := filepath.Join(s.Root, entry.StorePath)
		info, err := os.Lstat(storePath)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				os.Remove(storePath)
			} else {
				os.RemoveAll(storePath)
			}
		}

		// Remove from registry
		reg.Remove(name)
		if err := reg.Save(cfg); err != nil {
			return fmt.Errorf("saving registry: %w", err)
		}

		fmt.Printf("Removed %q.\n", name)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation")
	rootCmd.AddCommand(removeCmd)
}
