// Package cmd implements the ruleconv command-line interface (Cobra).
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ryliey/ruleconv/internal/catalog"
	"github.com/Ryliey/ruleconv/internal/client"
	"github.com/Ryliey/ruleconv/internal/compiler"
	"github.com/Ryliey/ruleconv/internal/config"
	"github.com/Ryliey/ruleconv/internal/sources"
	"github.com/Ryliey/ruleconv/internal/sync"
	"github.com/spf13/cobra"
)

var (
	flagRepo       string
	flagConfig     string
	flagSkipBinary bool
)

// version is injected at build time via
// -ldflags "-X github.com/Ryliey/ruleconv/cmd.version=v1.2.3", "dev" otherwise.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "ruleconv",
	Short: "Convert and synchronise proxy rule sets across clients",
	Long: `RuleConv converts a single authored "mixed" rule file into split (ip/site)
sources, compiles Clash (.mrs) and sing-box (.srs) binaries by shelling out to
the official cores, mirrors every service across all clients, and renders the
service and client README indexes from templates.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagRepo, "repo", ".", "path to the rules repository")
	pf.StringVar(&flagConfig, "config", "", "path to config.yaml (default <repo>/.ruleconv/config.yaml)")
	pf.BoolVar(&flagSkipBinary, "skip-binary", false, "do not invoke the cores; render sources and READMEs only")
}

// newEngine loads config + catalog and builds the sync engine, logging progress
// to stderr.
func newEngine() (*sync.Engine, error) {
	repoDir, err := filepath.Abs(flagRepo)
	if err != nil {
		return nil, err
	}

	cfgPath := flagConfig
	if cfgPath == "" {
		cfgPath = config.DefaultPath(repoDir)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	cat, err := catalog.Load(catalog.DefaultPath(repoDir))
	if err != nil {
		return nil, err
	}

	compiler.SetBinaries(cfg.Bin.Mihomo, cfg.Bin.Singbox)
	client.SetSingBoxVersion(cfg.SingBox.SourceVersion)

	eng := sync.New(repoDir, cfg, cat)
	eng.SkipBinary = flagSkipBinary
	eng.Logf = func(format string, a ...any) { fmt.Fprintf(os.Stderr, format+"\n", a...) }

	src, err := sources.Load(sources.DefaultPath(repoDir))
	if err != nil {
		return nil, err
	}
	eng.Sources = src
	return eng, nil
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
