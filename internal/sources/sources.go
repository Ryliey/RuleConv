// Package sources loads .ruleconv/sources.yaml: optional per-service upstream
// provenance (description + source URLs) cited in each service README.
package sources

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServiceMeta is one service's editorial provenance.
type ServiceMeta struct {
	Description string   `yaml:"description"`
	Sources     []string `yaml:"sources"`
}

// Sources is the parsed sources.yaml.
type Sources struct {
	Services map[string]ServiceMeta `yaml:"services"`
}

// DefaultPath returns the sources path inside repoDir.
func DefaultPath(repoDir string) string {
	return filepath.Join(repoDir, ".ruleconv", "sources.yaml")
}

// Load reads sources.yaml. A missing file yields empty sources.
func Load(path string) (*Sources, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Sources{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Sources
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &s, nil
}

// For returns the metadata for a service, or the zero value if absent.
func (s *Sources) For(service string) ServiceMeta {
	if s == nil {
		return ServiceMeta{}
	}
	return s.Services[service]
}
