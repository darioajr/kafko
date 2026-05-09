package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/darioajr/kafko/internal/config"
	"github.com/darioajr/kafko/internal/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
)

// resolveClientOptions merges global flags on top of the active profile.
// Flags take precedence over profile values; profile values are the fallback.
func resolveClientOptions() (kafka.ClientOptions, error) {
	cfg, err := config.Load(globalOpts.ConfigDir)
	if err != nil {
		return kafka.ClientOptions{}, err
	}
	prof, err := cfg.Resolve(globalOpts.Profile)
	if err != nil {
		return kafka.ClientOptions{}, err
	}
	return kafka.ClientOptions{
		Brokers:  pickStrings(globalOpts.Brokers, prof.Brokers),
		ClientID: prof.ClientID,
		Auth: kafka.AuthOptions{
			TLS:           globalOpts.TLS || prof.TLS,
			TLSCAFile:     pickString(globalOpts.TLSCA, prof.TLSCAFile),
			TLSCertFile:   pickString(globalOpts.TLSCert, prof.TLSCertFile),
			TLSKeyFile:    pickString(globalOpts.TLSKey, prof.TLSKeyFile),
			TLSInsecure:   globalOpts.TLSInsecure || prof.TLSInsecure,
			SASLMechanism: pickString(globalOpts.SASLMechanism, prof.SASLMechanism),
			SASLUsername:  pickString(globalOpts.SASLUsername, prof.SASLUsername),
			SASLPassword:  pickString(globalOpts.SASLPassword, prof.SASLPassword),
		},
	}, nil
}

func pickString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func pickStrings(a, b []string) []string {
	if len(a) > 0 {
		return a
	}
	return b
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func adminContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(globalOpts.Timeout)*time.Second)
}

// newAdminClient builds a *kadm.Client and a closer for the underlying kgo client.
func newAdminClient() (*kadm.Client, func(), error) {
	clientOpts, err := resolveClientOptions()
	if err != nil {
		return nil, nil, err
	}
	c, err := kafka.NewClient(clientOpts)
	if err != nil {
		return nil, nil, err
	}
	return kadm.NewClient(c), c.Close, nil
}
