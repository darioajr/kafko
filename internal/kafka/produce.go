package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

type ProducerOptions struct {
	RequiredAcks string // all|leader|none
	Compression  string // none|gzip|snappy|lz4|zstd
}

func NewProducer(client ClientOptions, p ProducerOptions) (*kgo.Client, error) {
	var extra []kgo.Opt

	switch p.RequiredAcks {
	case "", "all":
		extra = append(extra, kgo.RequiredAcks(kgo.AllISRAcks()))
	case "leader":
		extra = append(extra, kgo.RequiredAcks(kgo.LeaderAck()), kgo.DisableIdempotentWrite())
	case "none":
		extra = append(extra, kgo.RequiredAcks(kgo.NoAck()), kgo.DisableIdempotentWrite())
	default:
		return nil, fmt.Errorf("invalid acks %q (use all|leader|none)", p.RequiredAcks)
	}

	switch p.Compression {
	case "", "none":
		extra = append(extra, kgo.ProducerBatchCompression(kgo.NoCompression()))
	case "gzip":
		extra = append(extra, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	case "snappy":
		extra = append(extra, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	case "lz4":
		extra = append(extra, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	case "zstd":
		extra = append(extra, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	default:
		return nil, fmt.Errorf("invalid compression %q (use none|gzip|snappy|lz4|zstd)", p.Compression)
	}

	client.ExtraOpts = append(client.ExtraOpts, extra...)
	return NewClient(client)
}

// ProduceSync sends a single record and waits for the broker ack.
func ProduceSync(ctx context.Context, c *kgo.Client, r *kgo.Record) error {
	return c.ProduceSync(ctx, r).FirstErr()
}
