package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPickString(t *testing.T) {
	cases := []struct {
		a, b, want string
	}{
		{"", "", ""},
		{"", "fallback", "fallback"},
		{"override", "fallback", "override"},
		{"override", "", "override"},
	}
	for _, tc := range cases {
		if got := pickString(tc.a, tc.b); got != tc.want {
			t.Errorf("pickString(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPickStrings(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want []string
	}{
		{"both empty", nil, nil, nil},
		{"only fallback", nil, []string{"x"}, []string{"x"}},
		{"override wins", []string{"a"}, []string{"b"}, []string{"a"}},
		{"empty slice falls back", []string{}, []string{"b"}, []string{"b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pickStrings(tc.a, tc.b)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNewRootCmd_RegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	want := []string{"consume", "produce", "topics", "groups", "metadata", "tui", "version"}
	for _, name := range want {
		if _, _, err := root.Find([]string{name}); err != nil {
			t.Errorf("subcommand %q not registered: %v", name, err)
		}
	}
}

func TestNewRootCmd_HasGlobalFlags(t *testing.T) {
	root := NewRootCmd()
	want := []string{
		"brokers", "profile", "config-dir",
		"tls", "tls-ca", "tls-cert", "tls-key", "tls-insecure",
		"sasl-mechanism", "sasl-username", "sasl-password",
		"timeout", "verbose",
	}
	for _, name := range want {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("persistent flag %q not registered", name)
		}
	}
}

func TestVersionCommand_Output(t *testing.T) {
	t.Cleanup(func() { Version, Commit, Date = "dev", "none", "unknown" })
	Version, Commit, Date = "1.2.3", "abcdef0", "2026-01-01"

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	for _, frag := range []string{"kafko 1.2.3", "abcdef0", "2026-01-01"} {
		if !strings.Contains(got, frag) {
			t.Errorf("output missing %q\n---\n%s", frag, got)
		}
	}
}

func TestResolveClientOptions_FlagOverridesProfile(t *testing.T) {
	dir := t.TempDir()
	body := `default_profile = "p"

[profiles.p]
brokers = ["profile-broker:9092"]
tls = true
sasl_mechanism = "PLAIN"
sasl_username = "profile-user"
sasl_password = "profile-pass"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { *globalOpts = globalOptions{} })
	*globalOpts = globalOptions{
		ConfigDir:     dir,
		Brokers:       []string{"flag-broker:9092"},
		SASLMechanism: "SCRAM-SHA-512",
		SASLUsername:  "flag-user",
	}

	co, err := resolveClientOptions()
	if err != nil {
		t.Fatalf("resolveClientOptions: %v", err)
	}
	if len(co.Brokers) != 1 || co.Brokers[0] != "flag-broker:9092" {
		t.Errorf("brokers = %v, want [flag-broker:9092]", co.Brokers)
	}
	if co.Auth.SASLMechanism != "SCRAM-SHA-512" {
		t.Errorf("SASLMechanism = %q, want SCRAM-SHA-512", co.Auth.SASLMechanism)
	}
	if co.Auth.SASLUsername != "flag-user" {
		t.Errorf("SASLUsername = %q, want flag-user", co.Auth.SASLUsername)
	}
	if co.Auth.SASLPassword != "profile-pass" {
		t.Errorf("SASLPassword = %q, want profile-pass (fallback)", co.Auth.SASLPassword)
	}
	if !co.Auth.TLS {
		t.Errorf("TLS = false, want true (from profile)")
	}
}

func TestResolveClientOptions_ProfileOnly(t *testing.T) {
	dir := t.TempDir()
	body := `default_profile = "p"

[profiles.p]
brokers = ["b:9092"]
client_id = "kafko-test"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { *globalOpts = globalOptions{} })
	*globalOpts = globalOptions{ConfigDir: dir}

	co, err := resolveClientOptions()
	if err != nil {
		t.Fatalf("resolveClientOptions: %v", err)
	}
	if len(co.Brokers) != 1 || co.Brokers[0] != "b:9092" {
		t.Errorf("brokers = %v, want [b:9092]", co.Brokers)
	}
	if co.ClientID != "kafko-test" {
		t.Errorf("ClientID = %q, want kafko-test", co.ClientID)
	}
}

func TestParseHeaders(t *testing.T) {
	cases := []struct {
		name    string
		in      []string
		wantErr bool
		want    map[string]string
	}{
		{"nil input", nil, false, nil},
		{"empty input", []string{}, false, nil},
		{"single", []string{"a=1"}, false, map[string]string{"a": "1"}},
		{"multi", []string{"a=1", "b=2"}, false, map[string]string{"a": "1", "b": "2"}},
		{"empty value", []string{"a="}, false, map[string]string{"a": ""}},
		{"value with equals", []string{"a=x=y"}, false, map[string]string{"a": "x=y"}},
		{"missing separator", []string{"abc"}, true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := parseHeaders(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out) != len(tc.want) {
				t.Fatalf("len = %d, want %d", len(out), len(tc.want))
			}
			for _, h := range out {
				want, ok := tc.want[h.Key]
				if !ok {
					t.Errorf("unexpected header key %q", h.Key)
					continue
				}
				if string(h.Value) != want {
					t.Errorf("header[%q] = %q, want %q", h.Key, string(h.Value), want)
				}
			}
		})
	}
}

func TestBuildRecord(t *testing.T) {
	t.Run("value-only", func(t *testing.T) {
		f := &produceFlags{topic: "t", partition: -1}
		r := buildRecord([]byte("hello"), f, nil)
		if r.Topic != "t" || string(r.Value) != "hello" || r.Key != nil {
			t.Errorf("got topic=%q key=%q value=%q", r.Topic, r.Key, r.Value)
		}
	})

	t.Run("key with default tab separator", func(t *testing.T) {
		f := &produceFlags{topic: "t", includeKey: true, partition: -1}
		r := buildRecord([]byte("k1\tval"), f, nil)
		if string(r.Key) != "k1" || string(r.Value) != "val" {
			t.Errorf("got key=%q value=%q", r.Key, r.Value)
		}
	})

	t.Run("custom separator", func(t *testing.T) {
		f := &produceFlags{topic: "t", includeKey: true, keySep: ":", partition: -1}
		r := buildRecord([]byte("user-1:hello"), f, nil)
		if string(r.Key) != "user-1" || string(r.Value) != "hello" {
			t.Errorf("got key=%q value=%q", r.Key, r.Value)
		}
	})

	t.Run("includeKey but no separator falls through to value-only", func(t *testing.T) {
		f := &produceFlags{topic: "t", includeKey: true, keySep: ":", partition: -1}
		r := buildRecord([]byte("nokey"), f, nil)
		if r.Key != nil || string(r.Value) != "nokey" {
			t.Errorf("got key=%q value=%q", r.Key, r.Value)
		}
	})

	t.Run("forced partition", func(t *testing.T) {
		f := &produceFlags{topic: "t", partition: 7}
		r := buildRecord([]byte("x"), f, nil)
		if r.Partition != 7 {
			t.Errorf("partition = %d, want 7", r.Partition)
		}
	})
}

func TestResolveClientOptions_UnknownProfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { *globalOpts = globalOptions{} })
	*globalOpts = globalOptions{ConfigDir: dir, Profile: "nope"}

	if _, err := resolveClientOptions(); err == nil {
		t.Fatal("expected error for unknown profile")
	}
}
