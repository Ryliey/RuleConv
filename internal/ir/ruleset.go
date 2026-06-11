// Package ir defines the client-agnostic intermediate representation. Every
// client renders its files from a RuleSet and parses them back into one, so the
// IR is the canonical state of a service's rules.
package ir

import (
	"slices"
	"sort"
)

// RuleSet is the canonical representation of one service's rules: the domain
// dimension split into the four matcher kinds the cores distinguish, plus a
// flat list of IP CIDRs. See SiteOnly and IPOnly for the projections.
type RuleSet struct {
	Service       string
	Category      string
	Domain        []string
	DomainSuffix  []string
	DomainKeyword []string
	DomainRegex   []string
	IPCIDR        []string
}

// HasDomain reports whether any domain rule is present.
func (r RuleSet) HasDomain() bool {
	return len(r.Domain)+len(r.DomainSuffix)+len(r.DomainKeyword)+len(r.DomainRegex) > 0
}

// HasIP reports whether any IP-CIDR rule is present.
func (r RuleSet) HasIP() bool { return len(r.IPCIDR) > 0 }

// IsEmpty reports whether the rule set has no rules.
func (r RuleSet) IsEmpty() bool { return !r.HasDomain() && !r.HasIP() }

// SiteOnly returns a copy with only domain rules.
func (r RuleSet) SiteOnly() RuleSet {
	return RuleSet{
		Service:       r.Service,
		Category:      r.Category,
		Domain:        clone(r.Domain),
		DomainSuffix:  clone(r.DomainSuffix),
		DomainKeyword: clone(r.DomainKeyword),
		DomainRegex:   clone(r.DomainRegex),
	}
}

// IPOnly returns a copy with only IP-CIDR rules.
func (r RuleSet) IPOnly() RuleSet {
	return RuleSet{
		Service:  r.Service,
		Category: r.Category,
		IPCIDR:   clone(r.IPCIDR),
	}
}

// Merge returns the deduplicated union of r and o. Service/Category come from
// r, falling back to o when empty.
func (r RuleSet) Merge(o RuleSet) RuleSet {
	out := RuleSet{
		Service:       firstNonEmpty(r.Service, o.Service),
		Category:      firstNonEmpty(r.Category, o.Category),
		Domain:        append(clone(r.Domain), o.Domain...),
		DomainSuffix:  append(clone(r.DomainSuffix), o.DomainSuffix...),
		DomainKeyword: append(clone(r.DomainKeyword), o.DomainKeyword...),
		DomainRegex:   append(clone(r.DomainRegex), o.DomainRegex...),
		IPCIDR:        append(clone(r.IPCIDR), o.IPCIDR...),
	}
	return out.Dedup()
}

// Dedup returns a copy with each field sorted and deduplicated.
func (r RuleSet) Dedup() RuleSet {
	return RuleSet{
		Service:       r.Service,
		Category:      r.Category,
		Domain:        uniqSort(r.Domain),
		DomainSuffix:  uniqSort(r.DomainSuffix),
		DomainKeyword: uniqSort(r.DomainKeyword),
		DomainRegex:   uniqSort(r.DomainRegex),
		IPCIDR:        uniqSort(r.IPCIDR),
	}
}

// Equal reports whether two rule sets carry the same rules, ignoring Service
// and Category.
func Equal(a, b RuleSet) bool {
	a, b = a.Dedup(), b.Dedup()
	return slices.Equal(a.Domain, b.Domain) &&
		slices.Equal(a.DomainSuffix, b.DomainSuffix) &&
		slices.Equal(a.DomainKeyword, b.DomainKeyword) &&
		slices.Equal(a.DomainRegex, b.DomainRegex) &&
		slices.Equal(a.IPCIDR, b.IPCIDR)
}

func clone(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func uniqSort(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
