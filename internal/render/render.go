// Package render produces the Markdown READMEs. Templates are embedded via
// go:embed; a repo may override any of them by name in
// <repo>/.ruleconv/templates/. Overrides win over the embedded default.
package render

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.tmpl
var builtin embed.FS

// Template (and override) file names.
const (
	ServiceTemplate = "service_readme.tmpl"
	ClientTemplate  = "client_readme.tmpl"
)

// funcs are available to every template.
var funcs = template.FuncMap{
	"rows": chunkRows,
}

// chunkRows groups services into rows of n cells, padding the final row with
// empty ServiceRefs so every row is exactly n wide (for a uniform Markdown
// table).
func chunkRows(services []ServiceRef, n int) [][]ServiceRef {
	if n < 1 {
		n = 1
	}
	var out [][]ServiceRef
	for i := 0; i < len(services); i += n {
		row := make([]ServiceRef, n)
		copy(row, services[i:min(i+n, len(services))])
		out = append(out, row)
	}
	return out
}

// Renderer renders READMEs, resolving template overrides from OverrideDir.
type Renderer struct {
	// OverrideDir is typically <repo>/.ruleconv/templates. May be empty.
	OverrideDir string
}

// New returns a Renderer that looks for overrides in overrideDir.
func New(overrideDir string) *Renderer { return &Renderer{OverrideDir: overrideDir} }

// FileLink is one download URL for a generated file.
type FileLink struct {
	CDN string
	URL string
}

// FileEntry describes a generated file in a service directory.
type FileEntry struct {
	Name  string // e.g. "Google_site.mrs"
	Kind  string // "mixed" | "site" | "ip"
	Type  string // "source" | "binary"
	Links []FileLink
}

// ServiceData is the model for a service README.
type ServiceData struct {
	Client      string
	Service     string
	Category    string
	Description string
	Sources     []string
	Files       []FileEntry
}

// ServiceRef links to a service directory from the client index.
type ServiceRef struct {
	Name string
	Path string // e.g. "./Google"
}

// GroupData is one category section in a client README.
type GroupData struct {
	Name        string
	Description string
	Services    []ServiceRef
}

// ClientData is the model for a client README index.
type ClientData struct {
	Client string
	Owner  string
	Repo   string
	Branch string
	Groups []GroupData
}

// RenderService renders a service README.
func (r *Renderer) RenderService(data ServiceData) ([]byte, error) {
	return r.exec(ServiceTemplate, data)
}

// RenderClient renders a client index README.
func (r *Renderer) RenderClient(data ClientData) ([]byte, error) {
	return r.exec(ClientTemplate, data)
}

// EmbeddedTemplate returns the built-in template bytes.
func EmbeddedTemplate(name string) ([]byte, error) {
	return builtin.ReadFile("templates/" + name)
}

// EmbeddedTemplateNames lists the built-in templates.
func EmbeddedTemplateNames() []string {
	return []string{ServiceTemplate, ClientTemplate}
}

func (r *Renderer) exec(name string, data any) ([]byte, error) {
	t, err := r.load(name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *Renderer) load(name string) (*template.Template, error) {
	if r.OverrideDir != "" {
		if b, err := os.ReadFile(filepath.Join(r.OverrideDir, name)); err == nil {
			return template.New(name).Funcs(funcs).Parse(string(b))
		}
	}
	b, err := builtin.ReadFile("templates/" + name)
	if err != nil {
		return nil, err
	}
	return template.New(name).Funcs(funcs).Parse(string(b))
}
