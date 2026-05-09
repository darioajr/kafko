package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func newKfakeCluster(t *testing.T, topics ...string) *kfake.Cluster {
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

func runCLI(t *testing.T, brokers []string, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	t.Cleanup(func() { *globalOpts = globalOptions{} })

	root := NewRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetIn(strings.NewReader(stdin))

	full := append([]string{"--brokers", strings.Join(brokers, ",")}, args...)
	root.SetArgs(full)

	err = root.Execute()
	return out.String(), errBuf.String(), err
}

func TestCLI_TopicsCreateAndList(t *testing.T) {
	cluster := newKfakeCluster(t)
	addrs := cluster.ListenAddrs()

	out, _, err := runCLI(t, addrs, "", "topics", "create", "orders", "--partitions", "3")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out, `"orders" created`) {
		t.Errorf("unexpected create output: %q", out)
	}

	out, _, err = runCLI(t, addrs, "", "topics", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "orders") {
		t.Errorf("list output missing 'orders': %q", out)
	}
	if !strings.Contains(out, "PARTITIONS") {
		t.Errorf("list output missing header: %q", out)
	}
}

func TestCLI_TopicsDelete(t *testing.T) {
	const topic = "to-delete"
	cluster := newKfakeCluster(t, topic)
	addrs := cluster.ListenAddrs()

	if _, _, err := runCLI(t, addrs, "", "topics", "delete", topic); err != nil {
		t.Fatalf("delete: %v", err)
	}

	out, _, err := runCLI(t, addrs, "", "topics", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if strings.Contains(out, topic) {
		t.Errorf("topic still listed after delete: %q", out)
	}
}

func TestCLI_Produce_PipesStdinToTopic(t *testing.T) {
	const topic = "produced"
	cluster := newKfakeCluster(t, topic)
	addrs := cluster.ListenAddrs()

	stdin := "alpha\nbeta\ngamma\n"
	if _, _, err := runCLI(t, addrs, stdin, "produce", "-t", topic); err != nil {
		t.Fatalf("produce: %v", err)
	}

	c, err := kgo.NewClient(
		kgo.SeedBrokers(addrs...),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		t.Fatalf("verifier client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var values []string
	for len(values) < 3 {
		f := c.PollFetches(ctx)
		if errs := f.Errors(); len(errs) > 0 {
			t.Fatalf("fetch errors: %v", errs)
		}
		f.EachRecord(func(r *kgo.Record) {
			values = append(values, string(r.Value))
		})
	}
	want := []string{"alpha", "beta", "gamma"}
	for i, v := range want {
		if values[i] != v {
			t.Errorf("[%d] got %q, want %q", i, values[i], v)
		}
	}
}

func TestCLI_Consume_FromBeginningWithLimit(t *testing.T) {
	const topic = "to-consume"
	cluster := newKfakeCluster(t, topic)
	addrs := cluster.ListenAddrs()

	producer, err := kgo.NewClient(kgo.SeedBrokers(addrs...))
	if err != nil {
		t.Fatalf("seed producer: %v", err)
	}
	defer producer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, v := range []string{"one", "two"} {
		if err := producer.ProduceSync(ctx, &kgo.Record{Topic: topic, Value: []byte(v)}).FirstErr(); err != nil {
			t.Fatalf("ProduceSync %q: %v", v, err)
		}
	}

	out, _, err := runCLI(t, addrs, "",
		"consume", "-t", topic, "--from-beginning", "-n", "2")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") {
		t.Errorf("consume output missing values: %q", out)
	}
}

func TestCLI_Metadata(t *testing.T) {
	cluster := newKfakeCluster(t)
	addrs := cluster.ListenAddrs()

	out, _, err := runCLI(t, addrs, "", "metadata")
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	for _, frag := range []string{"Cluster:", "Controller:", "BROKER", "HOST", "PORT", "127.0.0.1"} {
		if !strings.Contains(out, frag) {
			t.Errorf("metadata output missing %q\n---\n%s", frag, out)
		}
	}
}
