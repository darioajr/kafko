package format

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/vmihailenco/msgpack/v5"
)

// msgpackFormatter decodes MessagePack record values to JSON for display.
// Falls back to base64 if the value cannot be decoded.
type msgpackFormatter struct{ opts Options }

func (f *msgpackFormatter) WriteRecord(w io.Writer, r *kgo.Record) error {
	var decoded any
	if err := msgpack.Unmarshal(r.Value, &decoded); err != nil {
		return fmt.Errorf("msgpack decode (offset %d): %w", r.Offset, err)
	}
	rec := jsonRecord{
		Topic:     r.Topic,
		Partition: r.Partition,
		Offset:    r.Offset,
		Timestamp: r.Timestamp.UnixMilli(),
		Key:       string(r.Key),
	}
	value, err := json.Marshal(decoded)
	if err != nil {
		return err
	}
	rec.Value = value
	if f.opts.Headers && len(r.Headers) > 0 {
		rec.Headers = make(map[string]string, len(r.Headers))
		for _, h := range r.Headers {
			rec.Headers[h.Key] = string(h.Value)
		}
	}
	return writeJSON(w, rec, f.opts.Pretty)
}
