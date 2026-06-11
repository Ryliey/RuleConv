package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Ryliey/ruleconv/internal/ir"
)

func sampleRuleSet() ir.RuleSet {
	return ir.RuleSet{
		Service:       "Google",
		Category:      "Global",
		Domain:        []string{"google.com"},
		DomainSuffix:  []string{"gstatic.com", "googleapis.com"},
		DomainKeyword: []string{"google"},
		DomainRegex:   []string{`^ad\..*\.google\.com$`},
		IPCIDR:        []string{"8.8.8.0/24", "2001:4860::/32"},
	}.Dedup()
}

func writeTemp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func assertDomainsEqual(t *testing.T, want, got ir.RuleSet) {
	t.Helper()
	if !reflect.DeepEqual(want.Domain, got.Domain) {
		t.Errorf("Domain: want %v got %v", want.Domain, got.Domain)
	}
	if !reflect.DeepEqual(want.DomainSuffix, got.DomainSuffix) {
		t.Errorf("DomainSuffix: want %v got %v", want.DomainSuffix, got.DomainSuffix)
	}
}

// Clash classical (Mixed) is lossless: it can express every matcher kind.
func TestClashMixedRoundTrip(t *testing.T) {
	c := Clash{}
	in := sampleRuleSet()
	b, err := c.RenderSource(in, Mixed)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Parse(writeTemp(t, "g.yaml", b), Mixed)
	if err != nil {
		t.Fatal(err)
	}
	want := in.SiteOnly().Merge(in.IPOnly()) // same fields, no Service/Category
	want.Service, want.Category = "", ""
	if !reflect.DeepEqual(want.Dedup(), got.Dedup()) {
		t.Errorf("Clash Mixed round-trip mismatch:\n want %+v\n got  %+v", want.Dedup(), got.Dedup())
	}
}

// Clash domain behaviour (Site) keeps exact + suffix; keyword/regex are
// intentionally dropped.
func TestClashSiteRoundTripDropsKeywordRegex(t *testing.T) {
	c := Clash{}
	in := sampleRuleSet()
	b, err := c.RenderSource(in, Site)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Parse(writeTemp(t, "g_site.yaml", b), Site)
	if err != nil {
		t.Fatal(err)
	}
	assertDomainsEqual(t, in, got)
	if got.HasIP() {
		t.Error("site must not contain IPs")
	}
	if len(got.DomainKeyword) != 0 || len(got.DomainRegex) != 0 {
		t.Error("domain behaviour cannot carry keyword/regex")
	}
}

func TestClashIPRoundTrip(t *testing.T) {
	c := Clash{}
	in := sampleRuleSet()
	b, err := c.RenderSource(in, IP)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Parse(writeTemp(t, "g_ip.yaml", b), IP)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in.IPCIDR, got.IPCIDR) {
		t.Errorf("IP round-trip: want %v got %v", in.IPCIDR, got.IPCIDR)
	}
}

func TestClashSupportsBinary(t *testing.T) {
	c := Clash{}
	if c.SupportsBinary(Mixed) {
		t.Error("Clash must NOT support a mixed/classical binary (.mrs has no classical behaviour)")
	}
	if !c.SupportsBinary(Site) || !c.SupportsBinary(IP) {
		t.Error("Clash should support domain/ipcidr binaries")
	}
}

// sing-box is lossless for every kind and may mix domain + IP in one set.
func TestSingBoxMixedRoundTrip(t *testing.T) {
	s := SingBox{}
	in := sampleRuleSet()
	b, err := s.RenderSource(in, Mixed)
	if err != nil {
		t.Fatal(err)
	}
	var probe sbRuleSet
	if err := json.Unmarshal(b, &probe); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if probe.Version != sourceVersion {
		t.Errorf("version: want %d got %d", sourceVersion, probe.Version)
	}
	got, err := s.Parse(writeTemp(t, "g.json", b), Mixed)
	if err != nil {
		t.Fatal(err)
	}
	want := in
	want.Service, want.Category = "", ""
	if !reflect.DeepEqual(want.Dedup(), got.Dedup()) {
		t.Errorf("sing-box round-trip mismatch:\n want %+v\n got  %+v", want.Dedup(), got.Dedup())
	}
}

// Guard the AND/OR semantics: each matcher field must be its own rule object.
func TestSingBoxEmitsSeparateRulesPerField(t *testing.T) {
	s := SingBox{}
	b, err := s.RenderSource(sampleRuleSet(), Mixed)
	if err != nil {
		t.Fatal(err)
	}
	var set sbRuleSet
	if err := json.Unmarshal(b, &set); err != nil {
		t.Fatal(err)
	}
	for i, r := range set.Rules {
		fields := 0
		for _, n := range []int{len(r.Domain), len(r.DomainSuffix), len(r.DomainKeyword), len(r.DomainRegex), len(r.IPCIDR)} {
			if n > 0 {
				fields++
			}
		}
		if fields != 1 {
			t.Errorf("rule %d has %d matcher fields; must be exactly 1 to keep OR semantics", i, fields)
		}
	}
}

func TestRegistry(t *testing.T) {
	if Get("Clash") == nil || Get("Sing-Box") == nil {
		t.Fatal("built-in clients must self-register")
	}
	names := strings.Join(Names(), ",")
	if !strings.Contains(names, "Clash") || !strings.Contains(names, "Sing-Box") {
		t.Errorf("registry names = %q", names)
	}
}
