package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert <service|path> [more...]",
	Short: "Convert/compile specific services (or paths) without a full pass",
	Long: `convert regenerates only the named services. Arguments containing a path
separator are treated as repo-relative paths and reconciled like sync --changed;
bare names are treated as service names and resolved from any client.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := newEngine()
		if err != nil {
			return err
		}
		var paths, services []string
		for _, a := range args {
			if strings.ContainsAny(a, "/\\") {
				paths = append(paths, a)
			} else {
				services = append(services, a)
			}
		}
		if len(paths) > 0 {
			if err := eng.SyncChanged(paths); err != nil {
				return err
			}
		}
		for _, s := range services {
			rs, ok, err := eng.ResolveService(s)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("service %q not found in any client", s)
			}
			rs.Service = s
			if err := eng.SyncService(rs); err != nil {
				return err
			}
		}
		return eng.RenderClientIndexes()
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}
