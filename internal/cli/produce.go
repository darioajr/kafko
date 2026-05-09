package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/darioajr/kafko/internal/kafka"
	"github.com/spf13/cobra"
	"github.com/twmb/franz-go/pkg/kgo"
)

type produceFlags struct {
	topic       string
	includeKey  bool
	keySep      string
	headers     []string
	acks        string
	compression string
	partition   int32
}

func newProduceCmd() *cobra.Command {
	f := &produceFlags{partition: -1, acks: "all"}
	cmd := &cobra.Command{
		Use:     "produce",
		Aliases: []string{"p"},
		Short:   "Produce messages to a topic from stdin (one record per line)",
		Example: `  echo "hello" | kafko produce -t orders
  cat events.jsonl | kafko produce -t events --acks=all
  kafko produce -t users -K --key-separator=":" <<< "u1:{\"name\":\"alice\"}"
  kafko produce -t logs -H trace-id=abc -H source=web < lines.txt`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if f.topic == "" {
				return fmt.Errorf("--topic is required")
			}

			ctx, cancel := signalContext()
			defer cancel()

			clientOpts, err := resolveClientOptions()
			if err != nil {
				return err
			}
			c, err := kafka.NewProducer(clientOpts, kafka.ProducerOptions{
				RequiredAcks: f.acks,
				Compression:  f.compression,
			})
			if err != nil {
				return err
			}
			defer c.Close()

			hdrs, err := parseHeaders(f.headers)
			if err != nil {
				return err
			}

			scanner := bufio.NewScanner(cmd.InOrStdin())
			scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
			var lines int
			for scanner.Scan() {
				rec := buildRecord(scanner.Bytes(), f, hdrs)
				if err := kafka.ProduceSync(ctx, c, rec); err != nil {
					return fmt.Errorf("produce: %w", err)
				}
				lines++
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			fmt.Fprintf(os.Stderr, "kafko: produced %d messages to %s\n", lines, f.topic)
			return nil
		},
	}

	pf := cmd.Flags()
	pf.StringVarP(&f.topic, "topic", "t", "", "topic to produce to (required)")
	pf.BoolVarP(&f.includeKey, "key", "K", false, "extract key from each line using --key-separator")
	pf.StringVar(&f.keySep, "key-separator", "\t", "separator splitting key and value in input")
	pf.StringSliceVarP(&f.headers, "header", "H", nil, "header in key=value form (repeatable)")
	pf.StringVar(&f.acks, "acks", "all", "required acks: all|leader|none")
	pf.StringVar(&f.compression, "compression", "", "compression: none|gzip|snappy|lz4|zstd")
	pf.Int32VarP(&f.partition, "partition", "p", -1, "force partition (default: hash by key)")

	return cmd
}

func buildRecord(line []byte, f *produceFlags, hdrs []kgo.RecordHeader) *kgo.Record {
	rec := &kgo.Record{Topic: f.topic, Headers: hdrs}
	if f.partition >= 0 {
		rec.Partition = f.partition
	}
	if f.includeKey {
		sep := f.keySep
		if sep == "" {
			sep = "\t"
		}
		idx := strings.Index(string(line), sep)
		if idx < 0 {
			rec.Value = append([]byte(nil), line...)
		} else {
			rec.Key = append([]byte(nil), line[:idx]...)
			rec.Value = append([]byte(nil), line[idx+len(sep):]...)
		}
	} else {
		rec.Value = append([]byte(nil), line...)
	}
	return rec
}

func parseHeaders(in []string) ([]kgo.RecordHeader, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]kgo.RecordHeader, 0, len(in))
	for _, s := range in {
		k, v, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header %q (expected key=value)", s)
		}
		out = append(out, kgo.RecordHeader{Key: k, Value: []byte(v)})
	}
	return out, nil
}
