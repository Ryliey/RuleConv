// Package repo models the rules repository on disk: discovering clients and
// services, mapping changed paths to services, and rebuilding a service's
// canonical IR from its source files.
package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ryliey/ruleconv/internal/client"
	"github.com/Ryliey/ruleconv/internal/ir"
)

// Target identifies the service a changed file belongs to.
type Target struct {
	Client  string
	Service string
	Kind    client.Kind
}

// ParsePath maps a repo-relative path (e.g. "Clash/Google/Google_site.yaml")
// to a Target. ok is false for non-source files (binaries, READMEs,
// catalog.yaml, .ruleconv/...).
func ParsePath(rel string) (Target, bool) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 3 {
		return Target{}, false
	}
	clientName, service, file := parts[0], parts[1], parts[2]
	c := client.Get(clientName)
	if c == nil {
		return Target{}, false
	}
	ext := c.SourceExt()
	if !strings.HasSuffix(file, ext) {
		return Target{}, false
	}
	base := strings.TrimSuffix(file, ext)
	kind := client.Mixed
	switch {
	case strings.HasSuffix(base, "_site"):
		kind = client.Site
	case strings.HasSuffix(base, "_ip"):
		kind = client.IP
	}
	return Target{Client: clientName, Service: service, Kind: kind}, true
}

// PresentClients returns registered clients that have a directory in repoDir.
func PresentClients(repoDir string) []client.Client {
	var out []client.Client
	for _, c := range client.All() {
		if isDir(filepath.Join(repoDir, c.Name())) {
			out = append(out, c)
		}
	}
	return out
}

// Services lists the sorted service directories under a client.
func Services(repoDir, clientName string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(repoDir, clientName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var svcs []string
	for _, e := range entries {
		if e.IsDir() {
			svcs = append(svcs, e.Name())
		}
	}
	sort.Strings(svcs)
	return svcs, nil
}

// AllServices returns the union of service names across all clients.
func AllServices(repoDir string) ([]string, error) {
	set := map[string]struct{}{}
	for _, c := range client.All() {
		svcs, err := Services(repoDir, c.Name())
		if err != nil {
			return nil, err
		}
		for _, s := range svcs {
			set[s] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}

// CanonicalIR builds a service's rule set from one client's files: the mixed
// source if present, else the union of the site and ip sources. found reports
// whether any source existed.
func CanonicalIR(repoDir, clientName, service string) (rs ir.RuleSet, found bool, err error) {
	c := client.Get(clientName)
	if c == nil {
		return ir.RuleSet{}, false, fmt.Errorf("unknown client %q", clientName)
	}
	dir := filepath.Join(repoDir, clientName, service)

	if mixed := filepath.Join(dir, service+c.SourceExt()); fileExists(mixed) {
		rs, err = c.Parse(mixed, client.Mixed)
		if err != nil {
			return ir.RuleSet{}, false, err
		}
		rs.Service = service
		return rs, true, nil
	}

	rs = ir.RuleSet{Service: service}
	for _, k := range []client.Kind{client.Site, client.IP} {
		p := filepath.Join(dir, service+k.Suffix()+c.SourceExt())
		if !fileExists(p) {
			continue
		}
		part, err := c.Parse(p, k)
		if err != nil {
			return ir.RuleSet{}, false, err
		}
		rs = rs.Merge(part)
		found = true
	}
	rs.Service = service
	return rs, found, nil
}

// ServiceSources returns each client's canonical IR for the service (keyed by
// client name) and the priority winner: the alphabetically-first client with a
// mixed source, else the first with any source. winner is "" when no client
// holds the service.
func ServiceSources(repoDir, service string) (map[string]ir.RuleSet, string, error) {
	sources := map[string]ir.RuleSet{}
	winner := ""
	for _, c := range client.All() {
		rs, ok, err := CanonicalIR(repoDir, c.Name(), service)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			continue
		}
		sources[c.Name()] = rs.Dedup()
		if winner == "" && fileExists(filepath.Join(repoDir, c.Name(), service, service+c.SourceExt())) {
			winner = c.Name()
		}
	}
	if winner == "" {
		for _, c := range client.All() {
			if _, ok := sources[c.Name()]; ok {
				winner = c.Name()
				break
			}
		}
	}
	return sources, winner, nil
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
