// Package catalog loads catalog.yaml: the service-to-category mapping and order
// for the per-client README index.
package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Uncategorized holds services on disk but missing from catalog.yaml.
const Uncategorized = "Uncategorized"

// Category is one ordered group of services.
type Category struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Services    []string `yaml:"services"`
}

// Catalog is the parsed catalog.yaml.
type Catalog struct {
	Categories []Category `yaml:"categories"`
}

// Group is a category with its existing services.
type Group struct {
	Name        string
	Description string
	Services    []string
}

// DefaultPath returns the catalog path inside repoDir.
func DefaultPath(repoDir string) string {
	return filepath.Join(repoDir, "catalog.yaml")
}

// Load reads catalog.yaml. A missing file yields an empty catalog.
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Catalog{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, nil
}

// CategoryOf returns the category name for a service, or Uncategorized.
func (c *Catalog) CategoryOf(service string) string {
	for _, cat := range c.Categories {
		for _, s := range cat.Services {
			if s == service {
				return cat.Name
			}
		}
	}
	return Uncategorized
}

// Grouped returns the catalog's categories filtered to existing services, in
// catalog order, with unlisted services appended under Uncategorized. Empty
// groups are dropped.
func (c *Catalog) Grouped(existing []string) []Group {
	have := make(map[string]bool, len(existing))
	for _, s := range existing {
		have[s] = true
	}
	used := make(map[string]bool)

	var groups []Group
	for _, cat := range c.Categories {
		var svcs []string
		for _, s := range cat.Services {
			if have[s] && !used[s] {
				svcs = append(svcs, s)
				used[s] = true
			}
		}
		if len(svcs) > 0 {
			groups = append(groups, Group{Name: cat.Name, Description: cat.Description, Services: svcs})
		}
	}

	var leftover []string
	for _, s := range existing {
		if !used[s] {
			leftover = append(leftover, s)
		}
	}
	if len(leftover) > 0 {
		groups = append(groups, Group{Name: Uncategorized, Services: leftover})
	}
	return groups
}
