package format

import (
	"fmt"
	"io"

	"github.com/twmb/franz-go/pkg/kgo"
)

type rawFormatter struct{ opts Options }

func (f *rawFormatter) WriteRecord(w io.Writer, r *kgo.Record) error {
	if f.opts.Metadata {
		if _, err := fmt.Fprintf(w, "%s/%d@%d ", r.Topic, r.Partition, r.Offset); err != nil {
			return err
		}
	}
	if f.opts.IncludeKey {
		if _, err := w.Write(r.Key); err != nil {
			return err
		}
		if _, err := io.WriteString(w, f.opts.KeySeparator); err != nil {
			return err
		}
	}
	if _, err := w.Write(r.Value); err != nil {
		return err
	}
	if f.opts.Headers && len(r.Headers) > 0 {
		if _, err := io.WriteString(w, "\t["); err != nil {
			return err
		}
		for i, h := range r.Headers {
			if i > 0 {
				if _, err := io.WriteString(w, ","); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "%s=%s", h.Key, h.Value); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "]"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}
