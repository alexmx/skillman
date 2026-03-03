package cmd

import (
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/alexmx/skillman/internal/config"
	"github.com/alexmx/skillman/internal/registry"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		reg, err := registry.Load(cfg)
		if err != nil {
			return err
		}

		if len(reg.Skills) == 0 {
			fmt.Println("No skills installed.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSOURCE\tREF\tINSTALLED")
		for _, e := range reg.Skills {
			ref := e.Ref
			if ref != "" && e.CommitSHA != "" {
				short := e.CommitSHA
				if len(short) > 7 {
					short = short[:7]
				}
				ref = ref + "@" + short
			} else if ref == "" {
				ref = "-"
			}
			src := e.Source
			if e.Local {
				src = "local"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name, src, ref, e.InstalledAt.Format("2006-01-02 15:04"))
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
