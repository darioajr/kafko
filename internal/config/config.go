// Package config loads kafko's TOML profile file.
//
// Layout (default $XDG_CONFIG_HOME/kafko/config.toml):
//
//	default_profile = "local"
//
//	[profiles.local]
//	brokers = ["localhost:9092"]
//
//	[profiles.prod]
//	brokers = ["kafka-0.prod:9093", "kafka-1.prod:9093"]
//	tls = true
//	sasl_mechanism = "SCRAM-SHA-512"
//	sasl_username = "kafko"
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Profile struct {
	Brokers       []string `toml:"brokers"`
	ClientID      string   `toml:"client_id"`
	TLS           bool     `toml:"tls"`
	TLSCAFile     string   `toml:"tls_ca_file"`
	TLSCertFile   string   `toml:"tls_cert_file"`
	TLSKeyFile    string   `toml:"tls_key_file"`
	TLSInsecure   bool     `toml:"tls_insecure"`
	SASLMechanism string   `toml:"sasl_mechanism"`
	SASLUsername  string   `toml:"sasl_username"`
	SASLPassword  string   `toml:"sasl_password"`
}

type File struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profiles"`
}

func DefaultDir() (string, error) {
	if d := os.Getenv("KAFKO_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "kafko"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kafko"), nil
}

// Load reads config.toml from dir. An empty dir falls back to DefaultDir.
// A missing file is not an error — Load returns an empty File.
func Load(dir string) (*File, error) {
	if dir == "" {
		var err error
		dir, err = DefaultDir()
		if err != nil {
			return nil, err
		}
	}
	path := filepath.Join(dir, "config.toml")
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{Profiles: map[string]Profile{}}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var f File
	if err := toml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	return &f, nil
}

// Resolve returns the profile to use given an explicit name.
// An empty name falls back to DefaultProfile, and if there is no default
// either, an empty profile is returned (no error).
func (f *File) Resolve(name string) (Profile, error) {
	if name == "" {
		name = f.DefaultProfile
	}
	if name == "" {
		return Profile{}, nil
	}
	p, ok := f.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}
