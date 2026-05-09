package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Missing(t *testing.T) {
	dir := t.TempDir()
	f, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(f.Profiles) != 0 {
		t.Fatalf("expected empty profiles, got %d", len(f.Profiles))
	}
}

func TestLoad_Parses(t *testing.T) {
	dir := t.TempDir()
	body := `default_profile = "local"

[profiles.local]
brokers = ["localhost:9092"]

[profiles.prod]
brokers = ["b1:9093", "b2:9093"]
tls = true
sasl_mechanism = "SCRAM-SHA-512"
sasl_username = "kafko"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if f.DefaultProfile != "local" {
		t.Fatalf("default_profile = %q", f.DefaultProfile)
	}
	prod, err := f.Resolve("prod")
	if err != nil {
		t.Fatal(err)
	}
	if len(prod.Brokers) != 2 || !prod.TLS || prod.SASLMechanism != "SCRAM-SHA-512" {
		t.Fatalf("unexpected prod profile: %+v", prod)
	}
}

func TestResolve_DefaultFallback(t *testing.T) {
	f := &File{
		DefaultProfile: "x",
		Profiles:       map[string]Profile{"x": {Brokers: []string{"a:9092"}}},
	}
	p, err := f.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Brokers) != 1 || p.Brokers[0] != "a:9092" {
		t.Fatalf("unexpected: %+v", p)
	}
}

func TestResolve_Unknown(t *testing.T) {
	f := &File{Profiles: map[string]Profile{}}
	if _, err := f.Resolve("nope"); err == nil {
		t.Fatal("expected error for unknown profile")
	}
}
