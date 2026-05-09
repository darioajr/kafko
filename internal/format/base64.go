package format

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/twmb/franz-go/pkg/kgo"
)

type b64Formatter struct{ opts Options }

func (f *b64Formatter) WriteRecord(w io.Writer, r *kgo.Record) error {
	if f.opts.Metadata {
		if _, err := fmt.Fprintf(w, "%s/%d@%d ", r.Topic, r.Partition, r.Offset); err != nil {
			return err
		}
	}
	enc := base64.StdEncoding
	if f.opts.IncludeKey {
		if _, err := io.WriteString(w, enc.EncodeToString(r.Key)+f.opts.KeySeparator); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, enc.EncodeToString(r.Value)+"\n")
	return err
}
