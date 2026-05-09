package cli

import (
	"fmt"
	"os"

	"github.com/darioajr/kafko/internal/format"
	"github.com/darioajr/kafko/internal/kafka"
	"github.com/spf13/cobra"
)

type consumeFlags struct {
	topics        []string
	group         string
	fromBeginning bool
	partition     int32
	offset        int64
	format        string
	pretty        bool
	includeKey    bool
	keySep        string
	headers       bool
	metadata      bool
	limit         int64
	protoFile     string
	protoMessage  string
}

func newConsumeCmd() *cobra.Command {
	f := &consumeFlags{partition: -1, offset: -1}
	cmd := &cobra.Command{
		Use:     "consume",
		Aliases: []string{"c"},
		Short:   "Consume messages from one or more topics",
		Example: `  kafko consume -t orders --from-beginning
  kafko consume -t orders -t payments -G my-app
  kafko consume -t orders -p 0 -o 1234 -n 10
  kafko consume -t orders -f json --pretty
  kafko consume -t binary -f hex
  kafko consume -t events -f msgpack --pretty
  kafko consume -t orders -f proto --proto-file=order.desc --proto-message=Order`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(f.topics) == 0 {
				return fmt.Errorf("at least one --topic is required")
			}

			ctx, cancel := signalContext()
			defer cancel()

			clientOpts, err := resolveClientOptions()
			if err != nil {
				return err
			}
			c, err := kafka.NewConsumer(clientOpts, kafka.ConsumeOptions{
				Topics:        f.topics,
				Group:         f.group,
				FromBeginning: f.fromBeginning,
				Partition:     f.partition,
				Offset:        f.offset,
			})
			if err != nil {
				return err
			}
			defer c.Close()

			fmtter, err := format.New(format.Options{
				Format:       f.format,
				Pretty:       f.pretty,
				IncludeKey:   f.includeKey,
				KeySeparator: f.keySep,
				Headers:      f.headers,
				Metadata:     f.metadata,
				ProtoFile:    f.protoFile,
				ProtoMessage: f.protoMessage,
			})
			if err != nil {
				return err
			}

			ch := make(chan kafka.Message, 64)
			go kafka.PollLoop(ctx, c, ch)

			out := cmd.OutOrStdout()
			var count int64
			for msg := range ch {
				if msg.Err != nil {
					fmt.Fprintf(os.Stderr, "kafko: %v\n", msg.Err)
					continue
				}
				if err := fmtter.WriteRecord(out, msg.Record); err != nil {
					return err
				}
				count++
				if f.limit > 0 && count >= f.limit {
					cancel()
				}
			}
			return nil
		},
	}

	pf := cmd.Flags()
	pf.StringSliceVarP(&f.topics, "topic", "t", nil, "topic(s) to consume from (repeatable)")
	pf.StringVarP(&f.group, "group", "G", "", "consumer group ID")
	pf.BoolVar(&f.fromBeginning, "from-beginning", false, "start at the earliest offset")
	pf.Int32VarP(&f.partition, "partition", "p", -1, "partition (use with --offset)")
	pf.Int64VarP(&f.offset, "offset", "o", -1, "explicit starting offset (use with --partition)")
	pf.StringVarP(&f.format, "format", "f", "raw", "output format: raw|json|hex|base64|msgpack|proto")
	pf.BoolVar(&f.pretty, "pretty", false, "pretty-print + colorize JSON output (json/msgpack/proto)")
	pf.BoolVarP(&f.includeKey, "key", "K", false, "include record key in output")
	pf.StringVar(&f.keySep, "key-separator", "\t", "separator between key and value")
	pf.BoolVarP(&f.headers, "headers", "H", false, "include record headers")
	pf.BoolVarP(&f.metadata, "metadata", "M", false, "prefix each line with topic/partition@offset")
	pf.Int64VarP(&f.limit, "limit", "n", 0, "stop after N messages (0 = unlimited)")
	pf.StringVar(&f.protoFile, "proto-file", "", "path to a .proto descriptor set (for -f proto)")
	pf.StringVar(&f.protoMessage, "proto-message", "", "fully-qualified protobuf message name (for -f proto)")

	return cmd
}
