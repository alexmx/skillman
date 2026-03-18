package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/alexmx/skillman/internal/workspace"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List skills in the current workspace",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		wc, err := workspace.LoadWorkspaceConfig(wd)
		if err != nil {
			return err
		}
		if wc == nil || len(wc.Skills) == 0 {
			fmt.Println("No skills in this workspace.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSOURCE\tREF")
		for _, e := range wc.Skills {
			ref := e.Ref
			if ref != "" && e.Commit != "" {
				ref = ref + "@" + shortSHA(e.Commit)
			} else if ref == "" {
				ref = "-"
			}
			src := e.Source
			if e.Path != "" {
				src = e.Path
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name, src, ref)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
