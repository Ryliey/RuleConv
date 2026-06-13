package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCDNURL(t *testing.T) {
	c := CDN{Name: "jsDelivr", Template: "https://cdn.jsdelivr.net/gh/{owner}/{repo}@{branch}/{path}"}
	got := c.URL("acme", "Rule", "main", "mihomo/Google/Google_site.mrs")
	want := "https://cdn.jsdelivr.net/gh/acme/Rule@main/mihomo/Google/Google_site.mrs"
	if got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

func TestRenderServiceEmbedded(t *testing.T) {
	r := New("")
	out, err := r.RenderService(ServiceData{
		Client:      "mihomo",
		Service:     "Google",
		Category:    "Global",
		Description: "Google core services",
		Sources:     []string{"https://example.test/goog.json", "https://example.test/domains"},
		Files: []FileEntry{{
			Name: "Google_site.mrs", Kind: "site", Type: "binary",
			Links: []FileLink{{CDN: "jsDelivr", URL: "https://example/Google_site.mrs"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"# Google · mihomo", "Global", "Google_site.mrs", "https://example/Google_site.mrs",
		"Google core services", "## Sources",
		"- https://example.test/goog.json", "- https://example.test/domains",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("service README missing %q\n---\n%s", want, s)
		}
	}
}

func TestRenderClientEmbedded(t *testing.T) {
	r := New("")
	out, err := r.RenderClient(ClientData{
		Client: "Sing-Box",
		Groups: []GroupData{{
			Name: "AI", Services: []ServiceRef{
				{Name: "Claude", Path: "./Claude"},
				{Name: "OpenAI", Path: "./OpenAI"},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"# Sing-Box Rules",
		"## AI",
		"| AI | | | | |",
		"| ------------- | ------------- | ------------- | ------------- | ------------- |",
		// Two services fill one row left-to-right; the row is padded to 5 cells.
		"| [Claude](./Claude/) | [OpenAI](./OpenAI/) | | |",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("client README missing %q\n---\n%s", want, s)
		}
	}
}

func TestOverrideTemplateWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ServiceTemplate), []byte("OVERRIDE {{.Service}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := New(dir)
	out, err := r.RenderService(ServiceData{Service: "Google"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != "OVERRIDE Google" {
		t.Errorf("override not used, got %q", got)
	}
}
