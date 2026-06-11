package cmd

import "github.com/spf13/cobra"

var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Re-render all README files from templates (no rule changes)",
	Long: `readme regenerates every service README and every client index from the
files currently on disk and the active templates (override or embedded). It does
not modify rule files or invoke the cores.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := newEngine()
		if err != nil {
			return err
		}
		return eng.RenderReadmes()
	},
}

func init() {
	rootCmd.AddCommand(readmeCmd)
}
