package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ryliey/ruleconv/internal/render"
	"github.com/spf13/cobra"
)

var templateForce bool

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage README templates",
}

var templateSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Materialise built-in templates into the repo and re-render READMEs",
	Long: `template sync writes the built-in (embedded) templates into
<repo>/.ruleconv/templates so they can be reviewed and customised, then
re-renders all READMEs. Existing template files are preserved unless --force is
given, so local customisations are not clobbered.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := newEngine()
		if err != nil {
			return err
		}
		dir := filepath.Join(eng.RepoDir, ".ruleconv", "templates")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for _, name := range render.EmbeddedTemplateNames() {
			b, err := render.EmbeddedTemplate(name)
			if err != nil {
				return err
			}
			dst := filepath.Join(dir, name)
			if templateForce || !fileExists(dst) {
				if err := os.WriteFile(dst, b, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "wrote %s\n", filepath.Join(".ruleconv", "templates", name))
			} else {
				fmt.Fprintf(os.Stderr, "kept existing %s (use --force to overwrite)\n", filepath.Join(".ruleconv", "templates", name))
			}
		}
		return eng.RenderReadmes()
	},
}

func init() {
	templateSyncCmd.Flags().BoolVar(&templateForce, "force", false, "overwrite existing template files")
	templateCmd.AddCommand(templateSyncCmd)
	rootCmd.AddCommand(templateCmd)
}
