package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func sampleRecord() *kgo.Record {
	return &kgo.Record{
		Topic:     "orders",
		Partition: 3,
		Offset:    42,
		Timestamp: time.UnixMilli(1700000000000),
		Key:       []byte("user-1"),
		Value:     []byte(`{"id":1}`),
		Headers: []kgo.RecordHeader{
			{Key: "trace-id", Value: []byte("abc")},
		},
	}
}

func TestRawFormatter_ValueOnly(t *testing.T) {
	f, err := New(Options{Format: "raw"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.WriteRecord(&buf, sampleRecord()); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != `{"id":1}`+"\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRawFormatter_KeyAndMetadata(t *testing.T) {
	f, err := New(Options{Format: "raw", IncludeKey: true, Metadata: true})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.WriteRecord(&buf, sampleRecord()); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "orders/3@42 ") {
		t.Fatalf("missing metadata prefix: %q", got)
	}
	if !strings.Contains(got, "user-1\t{\"id\":1}") {
		t.Fatalf("missing key/value: %q", got)
	}
}

func TestJSONFormatter(t *testing.T) {
	f, err := New(Options{Format: "json", Headers: true})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.WriteRecord(&buf, sampleRecord()); err != nil {
		t.Fatal(err)
	}
	var rec jsonRecord
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Topic != "orders" || rec.Partition != 3 || rec.Offset != 42 {
		t.Fatalf("metadata wrong: %+v", rec)
	}
	if rec.Key != "user-1" || string(rec.Value) != `{"id":1}` {
		t.Fatalf("payload wrong: %+v", rec)
	}
	if rec.Headers["trace-id"] != "abc" {
		t.Fatalf("headers wrong: %+v", rec.Headers)
	}
}

func TestHexFormatter(t *testing.T) {
	f, err := New(Options{Format: "hex"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.WriteRecord(&buf, &kgo.Record{Value: []byte{0xde, 0xad, 0xbe, 0xef}}); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "deadbeef\n" {
		t.Fatalf("got %q", got)
	}
}

func TestBase64Formatter(t *testing.T) {
	f, err := New(Options{Format: "base64"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.WriteRecord(&buf, &kgo.Record{Value: []byte("hi")}); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "aGk=\n" {
		t.Fatalf("got %q", got)
	}
}

func TestNew_Unknown(t *testing.T) {
	if _, err := New(Options{Format: "yaml"}); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
