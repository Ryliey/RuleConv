package client

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Ryliey/ruleconv/internal/compiler"
	"github.com/Ryliey/ruleconv/internal/ir"
)

// sourceVersion is the sing-box rule-set schema version written into generated
// .json (and thus the .srs). Version 4 means sing-box >= 1.13 on both ends.
// Override via config (singbox.sourceVersion).
var sourceVersion = 4

// SetSingBoxVersion overrides the sing-box source schema version.
func SetSingBoxVersion(v int) {
	if v > 0 {
		sourceVersion = v
	}
}

// SingBox renders rule sets for the sing-box core: JSON source, .srs binaries
// from `sing-box rule-set compile`. A single .srs may mix domain and IP rules.
type SingBox struct{}

func init() { Register(SingBox{}) }

func (SingBox) Name() string      { return "Sing-Box" }
func (SingBox) SourceExt() string { return ".json" }
func (SingBox) BinaryExt() string { return ".srs" }

func (SingBox) SupportsBinary(Kind) bool { return true }

type sbRule struct {
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
	IPCIDR        []string `json:"ip_cidr,omitempty"`
	Type          string   `json:"type,omitempty"`
}

type sbRuleSet struct {
	Version int      `json:"version"`
	Rules   []sbRule `json:"rules"`
}

func (SingBox) RenderSource(rs ir.RuleSet, k Kind) ([]byte, error) {
	set := sbRuleSet{Version: sourceVersion, Rules: []sbRule{}}
	add := func(r sbRule) { set.Rules = append(set.Rules, r) }

	// One rule per matcher field: rules are OR-ed, but multiple fields within a
	// single rule are AND-ed. Splitting them keeps the union semantics we want.
	if k == Mixed || k == Site {
		if len(rs.Domain) > 0 {
			add(sbRule{Domain: rs.Domain})
		}
		if len(rs.DomainSuffix) > 0 {
			add(sbRule{DomainSuffix: rs.DomainSuffix})
		}
		if len(rs.DomainKeyword) > 0 {
			add(sbRule{DomainKeyword: rs.DomainKeyword})
		}
		if len(rs.DomainRegex) > 0 {
			add(sbRule{DomainRegex: rs.DomainRegex})
		}
	}
	if k == Mixed || k == IP {
		if len(rs.IPCIDR) > 0 {
			add(sbRule{IPCIDR: rs.IPCIDR})
		}
	}

	out, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func (SingBox) Parse(path string, _ Kind) (ir.RuleSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ir.RuleSet{}, err
	}
	var set sbRuleSet
	if err := json.Unmarshal(data, &set); err != nil {
		return ir.RuleSet{}, fmt.Errorf("sing-box parse %s: %w", path, err)
	}
	rs := ir.RuleSet{}
	for _, r := range set.Rules {
		if r.Type == "logical" { // not representable in the flat IR
			continue
		}
		rs.Domain = append(rs.Domain, r.Domain...)
		rs.DomainSuffix = append(rs.DomainSuffix, r.DomainSuffix...)
		rs.DomainKeyword = append(rs.DomainKeyword, r.DomainKeyword...)
		rs.DomainRegex = append(rs.DomainRegex, r.DomainRegex...)
		rs.IPCIDR = append(rs.IPCIDR, r.IPCIDR...)
	}
	return rs.Dedup(), nil
}

func (SingBox) Compile(srcPath string, _ Kind, outPath string) error {
	return compiler.SingBoxCompile(srcPath, outPath)
}
