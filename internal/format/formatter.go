// Package format renders Kafka records into the output shapes kafko supports.
package format

import (
	"fmt"
	"io"

	"github.com/twmb/franz-go/pkg/kgo"
)

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

type Formatter interface {
	WriteRecord(w io.Writer, r *kgo.Record) error
}

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
