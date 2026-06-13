// Package client defines the pluggable adapter for a proxy core's rule-set
// formats and a registry of the built-in clients. To support a new core,
// implement Client and call Register from an init function.
package client

import (
	"sort"

	"github.com/Ryliey/ruleconv/internal/ir"
)

// Kind identifies which projection of a rule set a file represents.
type Kind int

const (
	Mixed Kind = iota // domains + IPs together
	Site              // domains only
	IP                // IP-CIDR only
)

// Kinds is the iteration order used when generating a service.
var Kinds = []Kind{Mixed, Site, IP}

// String returns the lowercase kind name.
func (k Kind) String() string {
	switch k {
	case Site:
		return "site"
	case IP:
		return "ip"
	default:
		return "mixed"
	}
}

// Suffix is the filename suffix for a kind; Mixed has none.
func (k Kind) Suffix() string {
	switch k {
	case Site:
		return "_site"
	case IP:
		return "_ip"
	default:
		return ""
	}
}

// Project returns the IR projection for kind k.
func Project(rs ir.RuleSet, k Kind) ir.RuleSet {
	switch k {
	case Site:
		return rs.SiteOnly()
	case IP:
		return rs.IPOnly()
	default:
		return rs
	}
}

// Client is a pluggable adapter for a proxy core's rule-set formats.
type Client interface {
	// Name is the directory name and display name, e.g. "mihomo".
	Name() string
	// SourceExt is the source file extension, e.g. ".yaml".
	SourceExt() string
	// BinaryExt is the compiled file extension, e.g. ".mrs".
	BinaryExt() string
	// SupportsBinary reports whether kind k can be compiled. mihomo cannot
	// compile a Mixed (classical) set.
	SupportsBinary(k Kind) bool
	// Parse reads a source file of the given kind into the canonical IR.
	Parse(path string, k Kind) (ir.RuleSet, error)
	// RenderSource renders the IR projection k into source bytes.
	RenderSource(rs ir.RuleSet, k Kind) ([]byte, error)
	// Compile compiles a source file of kind k into a binary at outPath.
	Compile(srcPath string, k Kind, outPath string) error
}

// Lossy is an optional Client interface reporting when an IR can't be fully
// represented in that client's formats (e.g. mihomo's domain behaviour drops
// keyword/regex matchers). Empty means no loss; the engine logs the rest as
// warnings.
type Lossy interface {
	LossyWarning(rs ir.RuleSet) string
}

var registry = map[string]Client{}

// Register adds a client to the global registry, from the client's init.
func Register(c Client) { registry[c.Name()] = c }

// Get returns the client registered under name, or nil if none.
func Get(name string) Client { return registry[name] }

// Names returns the registered client names, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// All returns all registered clients, sorted by name.
func All() []Client {
	out := make([]Client, 0, len(registry))
	for _, n := range Names() {
		out = append(out, registry[n])
	}
	return out
}

// SourceFileName builds the source filename for a service/kind on a client.
func SourceFileName(c Client, service string, k Kind) string {
	return service + k.Suffix() + c.SourceExt()
}

// BinaryFileName builds the binary filename for a service/kind on a client.
func BinaryFileName(c Client, service string, k Kind) string {
	return service + k.Suffix() + c.BinaryExt()
}
