package kafka

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func newCluster(t *testing.T, topics ...string) *kfake.Cluster {
	t.Helper()
	opts := []kfake.Opt{kfake.NumBrokers(1), kfake.AllowAutoTopicCreation()}
	if len(topics) > 0 {
		opts = append(opts, kfake.SeedTopics(1, topics...))
	}
	c, err := kfake.NewCluster(opts...)
	if err != nil {
		t.Fatalf("kfake.NewCluster: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestNewClient_NoBrokers(t *testing.T) {
	if _, err := NewClient(ClientOptions{}); err == nil {
		t.Fatal("expected error with empty Brokers")
	}
}

func TestNewClient_DefaultClientID(t *testing.T) {
	cluster := newCluster(t)
	c, err := NewClient(ClientOptions{Brokers: cluster.ListenAddrs()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()
	if got := c.OptValue(kgo.ClientID); got.(string) != "kafko" {
		t.Errorf("ClientID = %q, want kafko", got)
	}
}

func TestNewClient_CustomClientID(t *testing.T) {
	cluster := newCluster(t)
	c, err := NewClient(ClientOptions{Brokers: cluster.ListenAddrs(), ClientID: "custom"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()
	if got := c.OptValue(kgo.ClientID); got.(string) != "custom" {
		t.Errorf("ClientID = %q, want custom", got)
	}
}

func TestNewClient_BadSASLMechanism(t *testing.T) {
	_, err := NewClient(ClientOptions{
		Brokers: []string{"localhost:9092"},
		Auth: AuthOptions{
			SASLMechanism: "BOGUS",
			SASLUsername:  "u",
			SASLPassword:  "p",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported SASL") {
		t.Fatalf("expected unsupported SASL error, got %v", err)
	}
}

func TestNewClient_SASLMissingUsername(t *testing.T) {
	_, err := NewClient(ClientOptions{
		Brokers: []string{"localhost:9092"},
		Auth:    AuthOptions{SASLMechanism: "PLAIN"},
	})
	if err == nil || !strings.Contains(err.Error(), "sasl-username") {
		t.Fatalf("expected sasl-username error, got %v", err)
	}
}

func TestNewClient_MTLSRequiresBoth(t *testing.T) {
	_, err := NewClient(ClientOptions{
		Brokers: []string{"localhost:9092"},
		Auth:    AuthOptions{TLS: true, TLSCertFile: "cert.pem"},
	})
	if err == nil || !strings.Contains(err.Error(), "tls-cert") {
		t.Fatalf("expected mTLS pair error, got %v", err)
	}
}

func TestNewProducer_InvalidAcks(t *testing.T) {
	_, err := NewProducer(
		ClientOptions{Brokers: []string{"localhost:9092"}},
		ProducerOptions{RequiredAcks: "wat"},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid acks") {
		t.Fatalf("expected invalid acks error, got %v", err)
	}
}

func TestNewProducer_InvalidCompression(t *testing.T) {
	_, err := NewProducer(
		ClientOptions{Brokers: []string{"localhost:9092"}},
		ProducerOptions{Compression: "zlib"},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid compression") {
		t.Fatalf("expected invalid compression error, got %v", err)
	}
}

func TestProduceSync_Roundtrip(t *testing.T) {
	const topic = "kfake-roundtrip"
	cluster := newCluster(t, topic)
	addrs := cluster.ListenAddrs()

	producer, err := NewProducer(
		ClientOptions{Brokers: addrs},
		ProducerOptions{RequiredAcks: "all", Compression: "none"},
	)
	if err != nil {
		t.Fatalf("NewProducer: %v", err)
	}
	defer producer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ProduceSync(ctx, producer, &kgo.Record{
		Topic: topic,
		Key:   []byte("k1"),
		Value: []byte("v1"),
	}); err != nil {
		t.Fatalf("ProduceSync: %v", err)
	}

	consumer, err := NewClient(ClientOptions{
		Brokers:   addrs,
		ExtraOpts: []kgo.Opt{kgo.ConsumeTopics(topic), kgo.ConsumeResetOffset(kgo.NewOffset().AtStart())},
	})
	if err != nil {
		t.Fatalf("NewClient(consumer): %v", err)
	}
	defer consumer.Close()

	fetches := consumer.PollFetches(ctx)
	if errs := fetches.Errors(); len(errs) > 0 {
		t.Fatalf("fetch errors: %v", errs)
	}
	var seen int
	fetches.EachRecord(func(r *kgo.Record) {
		if string(r.Key) != "k1" || string(r.Value) != "v1" {
			t.Errorf("got key=%q value=%q, want k1/v1", r.Key, r.Value)
		}
		seen++
	})
	if seen != 1 {
		t.Errorf("got %d records, want 1", seen)
	}
}

func TestNewConsumer_NoTopics(t *testing.T) {
	if _, err := NewConsumer(ClientOptions{Brokers: []string{"x:9092"}}, ConsumeOptions{}); err == nil {
		t.Fatal("expected error for no topics")
	}
}

func TestNewConsumer_FromBeginning_PollLoop(t *testing.T) {
	const topic = "kfake-pollloop"
	cluster := newCluster(t, topic)
	addrs := cluster.ListenAddrs()

	producer, err := NewProducer(ClientOptions{Brokers: addrs}, ProducerOptions{})
	if err != nil {
		t.Fatalf("NewProducer: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < 3; i++ {
		if err := ProduceSync(ctx, producer, &kgo.Record{Topic: topic, Value: []byte("msg")}); err != nil {
			t.Fatalf("ProduceSync[%d]: %v", i, err)
		}
	}
	producer.Close()

	consumer, err := NewConsumer(
		ClientOptions{Brokers: addrs},
		ConsumeOptions{Topics: []string{topic}, FromBeginning: true},
	)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	defer consumer.Close()

	pollCtx, pollCancel := context.WithCancel(context.Background())
	defer pollCancel()
	ch := make(chan Message, 16)
	go PollLoop(pollCtx, consumer, ch)

	var got int
	timeout := time.After(5 * time.Second)
	for got < 3 {
		select {
		case msg, ok := <-ch:
			if !ok {
				t.Fatalf("channel closed early after %d", got)
			}
			if msg.Err != nil {
				t.Fatalf("poll error: %v", msg.Err)
			}
			if string(msg.Record.Value) != "msg" {
				t.Errorf("value = %q, want msg", msg.Record.Value)
			}
			got++
		case <-timeout:
			t.Fatalf("timed out after %d records", got)
		}
	}
}

func TestPollLoop_Cancellation(t *testing.T) {
	const topic = "kfake-cancel"
	cluster := newCluster(t, topic)

	consumer, err := NewConsumer(
		ClientOptions{Brokers: cluster.ListenAddrs()},
		ConsumeOptions{Topics: []string{topic}, FromBeginning: true},
	)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan Message, 4)
	done := make(chan struct{})
	go func() {
		PollLoop(ctx, consumer, ch)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("PollLoop did not exit on context cancel")
	}
	if _, ok := <-ch; ok {
		t.Error("expected ch closed on PollLoop exit")
	}
}
