// Package config loads <repo>/.ruleconv/config.yaml: repo coordinates, CDN
// providers, sing-box source version, and optional core binary paths.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the parsed .ruleconv/config.yaml.
type Config struct {
	Repo    RepoConfig    `yaml:"repo"`
	CDN     []CDNConfig   `yaml:"cdn"`
	SingBox SingBoxConfig `yaml:"singbox"`
	Bin     BinConfig     `yaml:"bin"`
	Cores   CoresConfig   `yaml:"cores"`
}

// RepoConfig identifies the GitHub repo, used to build download URLs.
type RepoConfig struct {
	Owner  string `yaml:"owner"`
	Name   string `yaml:"name"`
	Branch string `yaml:"branch"`
}

// CDNConfig is one download provider. Template placeholders: {owner} {repo}
// {branch} {path}.
type CDNConfig struct {
	Name     string `yaml:"name"`
	Template string `yaml:"template"`
}

// SingBoxConfig holds sing-box specific options.
type SingBoxConfig struct {
	SourceVersion int `yaml:"sourceVersion"`
}

// BinConfig holds explicit core binary paths. Empty values fall back to
// MIHOMO_BIN / SINGBOX_BIN, then PATH.
type BinConfig struct {
	Mihomo  string `yaml:"mihomo"`
	Singbox string `yaml:"singbox"`
}

// CoresConfig pins core versions for CI to download. RuleConv ignores these;
// only the workflows read them.
type CoresConfig struct {
	Mihomo  CoreVersion `yaml:"mihomo"`
	Singbox CoreVersion `yaml:"singbox"`
}

// CoreVersion pins a single core's version.
type CoreVersion struct {
	Version string `yaml:"version"`
}

// Default returns the built-in configuration with placeholder repo coordinates.
func Default() *Config {
	return &Config{
		Repo: RepoConfig{Owner: "your-org", Name: "your-repo", Branch: "main"},
		CDN: []CDNConfig{
			{Name: "GitHub", Template: "https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}"},
			{Name: "jsDelivr", Template: "https://cdn.jsdelivr.net/gh/{owner}/{repo}@{branch}/{path}"},
			{Name: "jsDelivr-CF", Template: "https://testingcf.jsdelivr.net/gh/{owner}/{repo}@{branch}/{path}"},
		},
		SingBox: SingBoxConfig{SourceVersion: 4},
	}
}

// DefaultPath returns the config path inside repoDir.
func DefaultPath(repoDir string) string {
	return filepath.Join(repoDir, ".ruleconv", "config.yaml")
}

// Load reads the config at path, layered over Default. A missing file returns
// the defaults.
func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	d := Default()
	if c.Repo.Branch == "" {
		c.Repo.Branch = d.Repo.Branch
	}
	if len(c.CDN) == 0 {
		c.CDN = d.CDN
	}
	if c.SingBox.SourceVersion == 0 {
		c.SingBox.SourceVersion = d.SingBox.SourceVersion
	}
}

// CDNs returns the configured providers, or the defaults if none.
func (c *Config) CDNs() []CDNConfig {
	if len(c.CDN) == 0 {
		return Default().CDN
	}
	return c.CDN
}
