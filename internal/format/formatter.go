// Package format renders Kafka records into the output shapes kafko supports.
package format

import (
	"fmt"
	"io"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Options selects an output format and tunes how each record is rendered.
// An empty KeySeparator defaults to "\t" inside New.
type Options struct {
	Format       string // raw|json|hex|base64|msgpack|proto
	Pretty       bool   // colorize+indent (json/msgpack/proto)
	IncludeKey   bool
	KeySeparator string
	Headers      bool
	Metadata     bool // prefix with topic/partition@offset

	// Proto-specific options.
	ProtoFile    string // path to a FileDescriptorSet (.desc / .pb)
	ProtoMessage string // fully-qualified message name
}

// Formatter writes a single Kafka record in the chosen output shape.
// Implementations must be safe for sequential calls from one goroutine.
type Formatter interface {
	WriteRecord(w io.Writer, r *kgo.Record) error
}

// New returns the Formatter that matches opts.Format. It returns an error
// for unknown formats or for "proto" when the descriptor file or message
// name is missing or invalid.
func New(opts Options) (Formatter, error) {
	if opts.KeySeparator == "" {
		opts.KeySeparator = "\t"
	}
	switch opts.Format {
	case "", "raw":
		return &rawFormatter{opts: opts}, nil
	case "json":
		return &jsonFormatter{opts: opts}, nil
	case "hex":
		return &hexFormatter{opts: opts}, nil
	case "base64":
		return &b64Formatter{opts: opts}, nil
	case "msgpack":
		return &msgpackFormatter{opts: opts}, nil
	case "proto":
		return newProtoFormatter(opts)
	default:
		return nil, fmt.Errorf("unsupported format %q (use raw|json|hex|base64|msgpack|proto)", opts.Format)
	}
}
