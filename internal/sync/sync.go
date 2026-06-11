// Package sync is the engine that converts and propagates rules. From one
// canonical IR it renders every client's sources, compiles binaries, writes the
// READMEs, and keeps all clients mirror-consistent.
package sync

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ryliey/ruleconv/internal/catalog"
	"github.com/Ryliey/ruleconv/internal/client"
	"github.com/Ryliey/ruleconv/internal/config"
	"github.com/Ryliey/ruleconv/internal/ir"
	"github.com/Ryliey/ruleconv/internal/render"
	"github.com/Ryliey/ruleconv/internal/repo"
)

// Engine performs conversions and propagation against a repository.
type Engine struct {
	RepoDir    string
	Cfg        *config.Config
	Cat        *catalog.Catalog
	Renderer   *render.Renderer
	SkipBinary bool                 // render sources + READMEs but do not invoke cores
	Logf       func(string, ...any) // progress sink; defaults to no-op
}

// New builds an Engine. repoDir should be absolute.
func New(repoDir string, cfg *config.Config, cat *catalog.Catalog) *Engine {
	return &Engine{
		RepoDir:  repoDir,
		Cfg:      cfg,
		Cat:      cat,
		Renderer: render.New(filepath.Join(repoDir, ".ruleconv", "templates")),
		Logf:     func(string, ...any) {},
	}
}

func (e *Engine) logf(format string, a ...any) {
	if e.Logf != nil {
		e.Logf(format, a...)
	}
}

// SyncAll resolves every service's canonical IR and regenerates everything.
func (e *Engine) SyncAll() error {
	services, err := repo.AllServices(e.RepoDir)
	if err != nil {
		return err
	}
	for _, s := range services {
		rs, ok, err := e.ResolveService(s)
		if err != nil {
			return err
		}
		if !ok || rs.IsEmpty() {
			e.logf("skip %s: no rules found", s)
			continue
		}
		rs.Service = s
		if err := e.SyncService(rs); err != nil {
			return err
		}
	}
	return e.RenderClientIndexes()
}

// SyncChanged reconciles only the services touched by the given paths. The
// edited client wins for that service, so editing a non-priority client takes
// effect instead of being reverted. A service with no remaining source is
// deleted from every client.
func (e *Engine) SyncChanged(paths []string) error {
	edited := map[string][]string{}
	for _, p := range paths {
		if t, ok := repo.ParsePath(e.relToRepo(p)); ok {
			edited[t.Service] = appendUnique(edited[t.Service], t.Client)
		}
	}
	if len(edited) == 0 {
		e.logf("no rule files among changed paths; regenerating indexes only")
		return e.RenderClientIndexes()
	}
	for s, clients := range edited {
		sort.Strings(clients)
		src := clients[0] // edited-client-wins; alphabetically first if several
		if len(clients) > 1 {
			e.logf("conflict: %q edited in %s — using %s as the source", s, strings.Join(clients, ", "), src)
		}
		rs, ok, err := repo.CanonicalIR(e.RepoDir, src, s)
		if err != nil {
			return err
		}
		if !ok || rs.IsEmpty() {
			e.logf("delete %s: no source remains in %s", s, src)
			if err := e.DeleteService(s); err != nil {
				return err
			}
			continue
		}
		rs.Service = s
		if err := e.SyncService(rs); err != nil {
			return err
		}
	}
	return e.RenderClientIndexes()
}

// ResolveService returns a service's canonical IR from the priority winner and
// warns when other clients hold diverging content the sync will overwrite. Used
// when there is no edited file to act as the authoritative source.
func (e *Engine) ResolveService(service string) (ir.RuleSet, bool, error) {
	sources, winner, err := repo.ServiceSources(e.RepoDir, service)
	if err != nil {
		return ir.RuleSet{}, false, err
	}
	if winner == "" {
		return ir.RuleSet{}, false, nil
	}
	win := sources[winner]
	var diverging []string
	for name, rs := range sources {
		if name != winner && !ir.Equal(rs, win) {
			diverging = append(diverging, name)
		}
	}
	if len(diverging) > 0 {
		sort.Strings(diverging)
		e.logf("conflict: %q differs across clients — keeping %s, overwriting %s", service, winner, strings.Join(diverging, ", "))
	}
	win.Service = service
	return win, true, nil
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

// SyncService regenerates one service across every registered client.
func (e *Engine) SyncService(rs ir.RuleSet) error {
	rs = rs.Dedup()
	if rs.Service == "" {
		return fmt.Errorf("rule set has no service name")
	}
	if rs.IsEmpty() {
		e.logf("skip %s: empty", rs.Service)
		return nil
	}
	rs.Category = e.Cat.CategoryOf(rs.Service)
	e.logf("sync %s [%s] (%d domain, %d ip)", rs.Service, rs.Category, domainCount(rs), len(rs.IPCIDR))

	for _, c := range client.All() {
		if err := e.writeServiceForClient(c, rs); err != nil {
			return fmt.Errorf("%s/%s: %w", c.Name(), rs.Service, err)
		}
	}
	return nil
}

func (e *Engine) writeServiceForClient(c client.Client, rs ir.RuleSet) error {
	dir := filepath.Join(e.RepoDir, c.Name(), rs.Service)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if l, ok := c.(client.Lossy); ok {
		if msg := l.LossyWarning(rs); msg != "" {
			e.logf("warning: %s/%s: %s", c.Name(), rs.Service, msg)
		}
	}

	var files []render.FileEntry
	for _, k := range client.Kinds {
		proj := client.Project(rs, k)
		if k != client.Mixed && proj.IsEmpty() {
			continue
		}

		srcName := client.SourceFileName(c, rs.Service, k)
		srcBytes, err := c.RenderSource(proj, k)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, srcName), srcBytes, 0o644); err != nil {
			return err
		}
		files = append(files, e.fileEntry(c, rs.Service, srcName, k, "source"))

		if !c.SupportsBinary(k) {
			continue
		}
		binName := client.BinaryFileName(c, rs.Service, k)
		binPath := filepath.Join(dir, binName)
		if e.SkipBinary {
			if fileExists(binPath) { // still advertise a pre-existing binary
				files = append(files, e.fileEntry(c, rs.Service, binName, k, "binary"))
			}
			continue
		}
		if err := c.Compile(filepath.Join(dir, srcName), k, binPath); err != nil {
			return fmt.Errorf("compile %s: %w", binName, err)
		}
		files = append(files, e.fileEntry(c, rs.Service, binName, k, "binary"))
	}

	return e.writeServiceReadme(c, rs.Service, rs.Category, files)
}

// DeleteService removes a service directory from every client.
func (e *Engine) DeleteService(service string) error {
	for _, c := range client.All() {
		dir := filepath.Join(e.RepoDir, c.Name(), service)
		if isDir(dir) {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			e.logf("removed %s", filepath.Join(c.Name(), service))
		}
	}
	return nil
}

// RenderReadmes re-renders all READMEs from the files on disk, without touching
// rule files or invoking the cores.
func (e *Engine) RenderReadmes() error {
	for _, c := range client.All() {
		services, err := repo.Services(e.RepoDir, c.Name())
		if err != nil {
			return err
		}
		for _, s := range services {
			if err := e.renderServiceReadmeFromDisk(c, s); err != nil {
				return err
			}
		}
	}
	return e.RenderClientIndexes()
}

// RenderClientIndexes regenerates the per-client category index README.
func (e *Engine) RenderClientIndexes() error {
	for _, c := range client.All() {
		clientDir := filepath.Join(e.RepoDir, c.Name())
		if !isDir(clientDir) {
			continue
		}
		services, err := repo.Services(e.RepoDir, c.Name())
		if err != nil {
			return err
		}
		if len(services) == 0 {
			continue
		}
		var groups []render.GroupData
		for _, g := range e.Cat.Grouped(services) {
			refs := make([]render.ServiceRef, 0, len(g.Services))
			for _, s := range g.Services {
				refs = append(refs, render.ServiceRef{Name: s, Path: "./" + s})
			}
			groups = append(groups, render.GroupData{Name: g.Name, Description: g.Description, Services: refs})
		}
		out, err := e.Renderer.RenderClient(render.ClientData{
			Client: c.Name(), Owner: e.Cfg.Repo.Owner, Repo: e.Cfg.Repo.Name, Branch: e.Cfg.Repo.Branch, Groups: groups,
		})
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(clientDir, "README.md"), out, 0o644); err != nil {
			return err
		}
		e.logf("index %s (%d services)", c.Name(), len(services))
	}
	return nil
}

// Validate parses every source file and reports those that fail.
func (e *Engine) Validate() error {
	problems := 0
	for _, c := range client.All() {
		services, err := repo.Services(e.RepoDir, c.Name())
		if err != nil {
			return err
		}
		for _, s := range services {
			for _, k := range client.Kinds {
				p := filepath.Join(e.RepoDir, c.Name(), s, client.SourceFileName(c, s, k))
				if !fileExists(p) {
					continue
				}
				if _, err := c.Parse(p, k); err != nil {
					e.logf("INVALID %s: %v", filepath.Join(c.Name(), s, client.SourceFileName(c, s, k)), err)
					problems++
				}
			}
		}
	}
	if problems > 0 {
		return fmt.Errorf("%d invalid rule file(s)", problems)
	}
	return nil
}

func (e *Engine) renderServiceReadmeFromDisk(c client.Client, service string) error {
	dir := filepath.Join(e.RepoDir, c.Name(), service)
	var files []render.FileEntry
	for _, k := range client.Kinds {
		if name := client.SourceFileName(c, service, k); fileExists(filepath.Join(dir, name)) {
			files = append(files, e.fileEntry(c, service, name, k, "source"))
		}
		if c.SupportsBinary(k) {
			if name := client.BinaryFileName(c, service, k); fileExists(filepath.Join(dir, name)) {
				files = append(files, e.fileEntry(c, service, name, k, "binary"))
			}
		}
	}
	return e.writeServiceReadme(c, service, e.Cat.CategoryOf(service), files)
}

func (e *Engine) writeServiceReadme(c client.Client, service, category string, files []render.FileEntry) error {
	out, err := e.Renderer.RenderService(render.ServiceData{
		Client: c.Name(), Service: service, Category: category, Files: files,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.RepoDir, c.Name(), service, "README.md"), out, 0o644)
}

func (e *Engine) fileEntry(c client.Client, service, fileName string, k client.Kind, typ string) render.FileEntry {
	rel := path.Join(c.Name(), service, fileName)
	links := make([]render.FileLink, 0, len(e.Cfg.CDNs()))
	for _, cd := range e.Cfg.CDNs() {
		u := render.CDN{Name: cd.Name, Template: cd.Template}.
			URL(e.Cfg.Repo.Owner, e.Cfg.Repo.Name, e.Cfg.Repo.Branch, rel)
		links = append(links, render.FileLink{CDN: cd.Name, URL: u})
	}
	return render.FileEntry{Name: fileName, Kind: k.String(), Type: typ, Links: links}
}

func (e *Engine) relToRepo(p string) string {
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(filepath.Clean(p))
	}
	base := e.RepoDir
	if abs, err := filepath.Abs(base); err == nil {
		base = abs
	}
	if r, err := filepath.Rel(base, p); err == nil && !strings.HasPrefix(r, "..") {
		return filepath.ToSlash(r)
	}
	return filepath.ToSlash(p)
}

func domainCount(rs ir.RuleSet) int {
	return len(rs.Domain) + len(rs.DomainSuffix) + len(rs.DomainKeyword) + len(rs.DomainRegex)
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
