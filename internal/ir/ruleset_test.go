package ir

import (
	"reflect"
	"testing"
)

func TestSiteIPSplit(t *testing.T) {
	rs := RuleSet{
		Service:       "Google",
		Domain:        []string{"google.com"},
		DomainSuffix:  []string{"gstatic.com"},
		DomainKeyword: []string{"google"},
		IPCIDR:        []string{"8.8.8.0/24"},
	}

	site := rs.SiteOnly()
	if site.HasIP() {
		t.Error("SiteOnly must not carry IP rules")
	}
	if !site.HasDomain() {
		t.Error("SiteOnly should carry domain rules")
	}
	if site.Service != "Google" {
		t.Errorf("SiteOnly should preserve Service, got %q", site.Service)
	}

	ipo := rs.IPOnly()
	if ipo.HasDomain() {
		t.Error("IPOnly must not carry domain rules")
	}
	if !reflect.DeepEqual(ipo.IPCIDR, []string{"8.8.8.0/24"}) {
		t.Errorf("IPOnly lost CIDRs: %v", ipo.IPCIDR)
	}
}

func TestMergeSortsAndDedups(t *testing.T) {
	a := RuleSet{DomainSuffix: []string{"b.com", "a.com", "a.com"}}
	b := RuleSet{DomainSuffix: []string{"a.com", "c.com"}, IPCIDR: []string{"1.1.1.0/24"}}

	m := a.Merge(b)
	if !reflect.DeepEqual(m.DomainSuffix, []string{"a.com", "b.com", "c.com"}) {
		t.Errorf("Merge should sort+dedup, got %v", m.DomainSuffix)
	}
	if !reflect.DeepEqual(m.IPCIDR, []string{"1.1.1.0/24"}) {
		t.Errorf("Merge should keep IPs, got %v", m.IPCIDR)
	}
}

func TestIsEmpty(t *testing.T) {
	if !(RuleSet{}).IsEmpty() {
		t.Error("zero RuleSet should be empty")
	}
	if (RuleSet{Domain: []string{"x"}}).IsEmpty() {
		t.Error("RuleSet with a domain is not empty")
	}
	if (RuleSet{IPCIDR: []string{"1.0.0.0/8"}}).IsEmpty() {
		t.Error("RuleSet with an IP is not empty")
	}
}

func TestDedupDropsEmptyStrings(t *testing.T) {
	r := RuleSet{Domain: []string{"", "x.com", ""}}.Dedup()
	if !reflect.DeepEqual(r.Domain, []string{"x.com"}) {
		t.Errorf("Dedup should drop empty strings, got %v", r.Domain)
	}
}

func TestEqualIgnoresServiceCategoryAndOrder(t *testing.T) {
	a := RuleSet{Service: "A", Category: "X", DomainSuffix: []string{"b.com", "a.com"}, IPCIDR: []string{"1.0.0.0/8"}}
	b := RuleSet{Service: "B", Category: "Y", DomainSuffix: []string{"a.com", "b.com"}, IPCIDR: []string{"1.0.0.0/8"}}
	if !Equal(a, b) {
		t.Error("Equal must ignore Service/Category and ordering")
	}
	if Equal(a, RuleSet{DomainSuffix: []string{"a.com"}}) {
		t.Error("rule sets with different rules must not be equal")
	}
	if !Equal(RuleSet{}, RuleSet{}) {
		t.Error("two empty rule sets are equal")
	}
}
