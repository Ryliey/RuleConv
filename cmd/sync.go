package cmd

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	syncAll     bool
	syncChanged []string
)

var syncCmd = &cobra.Command{
	Use:   "sync [paths...]",
	Short: "Convert, split, compile and propagate rules across clients",
	Long: `sync regenerates services. With no arguments (or --all) it rebuilds every
service. Given changed/deleted paths (via --changed or as positional arguments)
it reconciles only the affected services — deleting any whose sources no longer
exist. A change to catalog.yaml or .ruleconv/ triggers a full pass.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := newEngine()
		if err != nil {
			return err
		}
		changed := append(append([]string{}, syncChanged...), args...)
		if syncAll || len(changed) == 0 || needsFullPass(changed) {
			return eng.SyncAll()
		}
		return eng.SyncChanged(changed)
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "regenerate every service")
	syncCmd.Flags().StringSliceVar(&syncChanged, "changed", nil, "repo-relative changed/deleted paths to reconcile")
	rootCmd.AddCommand(syncCmd)
}

// needsFullPass reports whether a changed path touches global state
// (catalog.yaml or .ruleconv/), which warrants a full regeneration.
func needsFullPass(paths []string) bool {
	for _, p := range paths {
		s := filepath.ToSlash(p)
		if strings.HasSuffix(s, "catalog.yaml") || strings.Contains(s, ".ruleconv/") {
			return true
		}
	}
	return false
}
