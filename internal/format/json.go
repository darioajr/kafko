package format

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/term"
)

type jsonFormatter struct{ opts Options }

type jsonRecord struct {
	Topic     string            `json:"topic"`
	Partition int32             `json:"partition"`
	Offset    int64             `json:"offset"`
	Timestamp int64             `json:"timestamp_ms"`
	Key       string            `json:"key,omitempty"`
	Value     json.RawMessage   `json:"value"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func (f *jsonFormatter) WriteRecord(w io.Writer, r *kgo.Record) error {
	rec := jsonRecord{
		Topic:     r.Topic,
		Partition: r.Partition,
		Offset:    r.Offset,
		Timestamp: r.Timestamp.UnixMilli(),
		Key:       string(r.Key),
		Value:     valueAsJSON(r.Value),
	}
	if f.opts.Headers && len(r.Headers) > 0 {
		rec.Headers = make(map[string]string, len(r.Headers))
		for _, h := range r.Headers {
			rec.Headers[h.Key] = string(h.Value)
		}
	}
	return writeJSON(w, rec, f.opts.Pretty)
}

// valueAsJSON returns the raw value as JSON if it parses; otherwise as a string.
func valueAsJSON(v []byte) json.RawMessage {
	if json.Valid(v) {
		return json.RawMessage(v)
	}
	q, _ := json.Marshal(string(v))
	return q
}

func writeJSON(w io.Writer, v any, pretty bool) error {
	if !pretty {
		return json.NewEncoder(w).Encode(v)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	if shouldColorize(w) {
		return quick.Highlight(w, buf.String(), "json", "terminal16m", "monokai")
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// shouldColorize returns true when the writer is a TTY and the user did not
// disable colors via NO_COLOR (https://no-color.org).
func shouldColorize(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	type fder interface{ Fd() uintptr }
	f, ok := w.(fder)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
