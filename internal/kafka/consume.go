package kafka

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"
)

type ConsumeOptions struct {
	Topics        []string
	Group         string
	FromBeginning bool
	Partition     int32 // -1 = unset
	Offset        int64 // -1 = unset
}

// NewConsumer builds a *kgo.Client wired for consumption. The caller closes it.
func NewConsumer(client ClientOptions, c ConsumeOptions) (*kgo.Client, error) {
	if len(c.Topics) == 0 {
		return nil, errors.New("no topics specified")
	}

	var extra []kgo.Opt

	switch {
	case c.Partition >= 0 && c.Offset >= 0:
		// Pin to a single partition + offset.
		partOff := map[string]map[int32]kgo.Offset{
			c.Topics[0]: {c.Partition: kgo.NewOffset().At(c.Offset)},
		}
		extra = append(extra, kgo.ConsumePartitions(partOff))
	default:
		extra = append(extra, kgo.ConsumeTopics(c.Topics...))
		if c.FromBeginning {
			extra = append(extra, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
		} else {
			extra = append(extra, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
		}
	}

	if c.Group != "" {
		extra = append(extra, kgo.ConsumerGroup(c.Group), kgo.DisableAutoCommit())
	}

	client.ExtraOpts = append(client.ExtraOpts, extra...)
	return NewClient(client)
}

type Message struct {
	Record *kgo.Record
	Err    error
}

// PollLoop pumps fetches into out until ctx is cancelled. Closes out on exit.
func PollLoop(ctx context.Context, c *kgo.Client, out chan<- Message) {
	defer close(out)
	for {
		fetches := c.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		fetches.EachError(func(_ string, _ int32, err error) {
			select {
			case out <- Message{Err: err}:
			case <-ctx.Done():
			}
		})
		fetches.EachRecord(func(r *kgo.Record) {
			select {
			case out <- Message{Record: r}:
			case <-ctx.Done():
			}
		})
	}
}
