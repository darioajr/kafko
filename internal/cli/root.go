// Package cli wires kafko's cobra command tree.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables, populated via -ldflags by goreleaser / make.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type globalOptions struct {
	Brokers       []string
	Profile       string
	ConfigDir     string
	TLS           bool
	TLSCA         string
	TLSCert       string
	TLSKey        string
	TLSInsecure   bool
	SASLMechanism string
	SASLUsername  string
	SASLPassword  string
	Timeout       int
	Verbose       bool
}

var globalOpts = &globalOptions{}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kafko",
		Short: "kafko — speak Kafka from your terminal",
		Long: `kafko is a modern, container-friendly Kafka CLI.

Produce, consume, and inspect Kafka topics with a Unix pipe philosophy.
No JVM, no librdkafka — just a single static binary.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date),
	}

	pf := cmd.PersistentFlags()
	pf.StringSliceVarP(&globalOpts.Brokers, "brokers", "b", nil, "Kafka bootstrap brokers (comma-separated)")
	pf.StringVarP(&globalOpts.Profile, "profile", "c", "", "config profile name")
	pf.StringVar(&globalOpts.ConfigDir, "config-dir", "", "override config directory (default: $XDG_CONFIG_HOME/kafko)")
	pf.BoolVar(&globalOpts.TLS, "tls", false, "enable TLS")
	pf.StringVar(&globalOpts.TLSCA, "tls-ca", "", "CA certificate file")
	pf.StringVar(&globalOpts.TLSCert, "tls-cert", "", "client certificate file (mTLS)")
	pf.StringVar(&globalOpts.TLSKey, "tls-key", "", "client key file (mTLS)")
	pf.BoolVar(&globalOpts.TLSInsecure, "tls-insecure", false, "skip TLS verification (DANGEROUS)")
	pf.StringVar(&globalOpts.SASLMechanism, "sasl-mechanism", "", "SASL mechanism: PLAIN|SCRAM-SHA-256|SCRAM-SHA-512")
	pf.StringVar(&globalOpts.SASLUsername, "sasl-username", "", "SASL username")
	pf.StringVar(&globalOpts.SASLPassword, "sasl-password", "", "SASL password (or KAFKO_SASL_PASSWORD env)")
	pf.IntVar(&globalOpts.Timeout, "timeout", 30, "command timeout in seconds (admin ops)")
	pf.BoolVarP(&globalOpts.Verbose, "verbose", "v", false, "verbose output")

	cmd.AddCommand(
		newConsumeCmd(),
		newProduceCmd(),
		newTopicsCmd(),
		newGroupsCmd(),
		newMetadataCmd(),
		newTUICmd(),
		newVersionCmd(),
	)

	return cmd
}

func Execute() error {
	return NewRootCmd().Execute()
}
