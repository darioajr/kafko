package format

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/twmb/franz-go/pkg/kgo"
)

type hexFormatter struct{ opts Options }

func (f *hexFormatter) WriteRecord(w io.Writer, r *kgo.Record) error {
	if f.opts.Metadata {
		if _, err := fmt.Fprintf(w, "%s/%d@%d ", r.Topic, r.Partition, r.Offset); err != nil {
			return err
		}
	}
	if f.opts.IncludeKey {
		if _, err := io.WriteString(w, hex.EncodeToString(r.Key)+f.opts.KeySeparator); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, hex.EncodeToString(r.Value)+"\n")
	return err
}
