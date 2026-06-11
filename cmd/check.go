package cmd

import "github.com/spf13/cobra"

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate every rule file, then fully resync",
	Long: `check parses every source file to confirm it is well-formed, then runs a
full sync. This is the workflow used after adding a new client (or any large
change): it validates inputs and brings every client into a consistent state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := newEngine()
		if err != nil {
			return err
		}
		if err := eng.Validate(); err != nil {
			return err
		}
		return eng.SyncAll()
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
