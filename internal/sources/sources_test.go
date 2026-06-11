package sources

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	yaml := `services:
  Google:
    description: Google core services
    sources:
      - https://example.test/a
      - https://example.test/b
`
	p := filepath.Join(t.TempDir(), "sources.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	got := s.For("Google")
	if got.Description != "Google core services" {
		t.Errorf("description = %q", got.Description)
	}
	if !reflect.DeepEqual(got.Sources, []string{"https://example.test/a", "https://example.test/b"}) {
		t.Errorf("sources = %v", got.Sources)
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if got := s.For("Anything"); got.Description != "" || len(got.Sources) != 0 {
		t.Errorf("absent service should be zero, got %+v", got)
	}
}

func TestForNilSafe(t *testing.T) {
	var s *Sources
	if got := s.For("X"); got.Description != "" || len(got.Sources) != 0 {
		t.Errorf("nil receiver should return zero, got %+v", got)
	}
}
