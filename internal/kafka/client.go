// Package kafka wraps franz-go with the option set kafko exposes on the CLI.
package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type AuthOptions struct {
	TLS         bool
	TLSCAFile   string
	TLSCertFile string
	TLSKeyFile  string
	TLSInsecure bool

	SASLMechanism string
	SASLUsername  string
	SASLPassword  string
}

type ClientOptions struct {
	Brokers   []string
	ClientID  string
	Auth      AuthOptions
	ExtraOpts []kgo.Opt
}

func NewClient(opts ClientOptions) (*kgo.Client, error) {
	if len(opts.Brokers) == 0 {
		return nil, errors.New("no brokers configured (use --brokers or a profile)")
	}

	clientID := opts.ClientID
	if clientID == "" {
		clientID = "kafko"
	}

	kopts := []kgo.Opt{
		kgo.SeedBrokers(opts.Brokers...),
		kgo.ClientID(clientID),
	}

	tlsCfg, err := buildTLS(opts.Auth)
	if err != nil {
		return nil, fmt.Errorf("tls: %w", err)
	}
	if tlsCfg != nil {
		kopts = append(kopts, kgo.DialTLSConfig(tlsCfg))
	}

	saslMech, err := buildSASL(opts.Auth)
	if err != nil {
		return nil, fmt.Errorf("sasl: %w", err)
	}
	if saslMech != nil {
		kopts = append(kopts, kgo.SASL(saslMech))
	}

	kopts = append(kopts, opts.ExtraOpts...)

	return kgo.NewClient(kopts...)
}

func buildTLS(a AuthOptions) (*tls.Config, error) {
	if !a.TLS && a.TLSCAFile == "" && a.TLSCertFile == "" && a.TLSKeyFile == "" {
		return nil, nil
	}
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: a.TLSInsecure,
	}
	if a.TLSCAFile != "" {
		ca, err := os.ReadFile(a.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, errors.New("invalid CA certificate")
		}
		cfg.RootCAs = pool
	}
	if a.TLSCertFile != "" || a.TLSKeyFile != "" {
		if a.TLSCertFile == "" || a.TLSKeyFile == "" {
			return nil, errors.New("both --tls-cert and --tls-key are required for mTLS")
		}
		cert, err := tls.LoadX509KeyPair(a.TLSCertFile, a.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load keypair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

func buildSASL(a AuthOptions) (sasl.Mechanism, error) {
	if a.SASLMechanism == "" {
		return nil, nil
	}
	if a.SASLUsername == "" {
		return nil, errors.New("--sasl-username is required when SASL is enabled")
	}
	password := a.SASLPassword
	if password == "" {
		password = os.Getenv("KAFKO_SASL_PASSWORD")
	}
	switch a.SASLMechanism {
	case "PLAIN":
		return plain.Auth{User: a.SASLUsername, Pass: password}.AsMechanism(), nil
	case "SCRAM-SHA-256":
		return scram.Auth{User: a.SASLUsername, Pass: password}.AsSha256Mechanism(), nil
	case "SCRAM-SHA-512":
		return scram.Auth{User: a.SASLUsername, Pass: password}.AsSha512Mechanism(), nil
	default:
		return nil, fmt.Errorf("unsupported SASL mechanism %q (use PLAIN|SCRAM-SHA-256|SCRAM-SHA-512)", a.SASLMechanism)
	}
}
