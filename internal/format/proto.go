package format

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// protoFormatter decodes record values using a Protobuf FileDescriptorSet.
//
// Generate the descriptor with:
//
//	protoc --include_imports --descriptor_set_out=order.desc order.proto
type protoFormatter struct {
	opts    Options
	msgType protoreflect.MessageType
}

func newProtoFormatter(opts Options) (Formatter, error) {
	if opts.ProtoFile == "" {
		return nil, errors.New("--proto-file is required for -f proto")
	}
	if opts.ProtoMessage == "" {
		return nil, errors.New("--proto-message is required for -f proto (e.g. com.example.Order)")
	}
	mt, err := loadProtoMessage(opts.ProtoFile, opts.ProtoMessage)
	if err != nil {
		return nil, err
	}
	return &protoFormatter{opts: opts, msgType: mt}, nil
}

func loadProtoMessage(descPath, fqn string) (protoreflect.MessageType, error) {
	raw, err := os.ReadFile(descPath)
	if err != nil {
		return nil, fmt.Errorf("read descriptor: %w", err)
	}
	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &fds); err != nil {
		return nil, fmt.Errorf("parse descriptor: %w", err)
	}
	files, err := protodesc.NewFiles(&fds)
	if err != nil {
		return nil, fmt.Errorf("build descriptor index: %w", err)
	}
	desc, err := files.FindDescriptorByName(protoreflect.FullName(fqn))
	if err != nil {
		return nil, fmt.Errorf("find message %q: %w", fqn, err)
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%q is not a message", fqn)
	}
	// Use a registry-backed type so unknown nested types also resolve.
	types := dynamicpb.NewTypes(files)
	mt, err := types.FindMessageByName(md.FullName())
	if err != nil {
		// Fall back to a bare dynamic type if the registry lookup fails.
		if errors.Is(err, protoregistry.NotFound) {
			return dynamicpb.NewMessageType(md), nil
		}
		return nil, err
	}
	return mt, nil
}

func (f *protoFormatter) WriteRecord(w io.Writer, r *kgo.Record) error {
	msg := f.msgType.New()
	if err := proto.Unmarshal(r.Value, msg.Interface()); err != nil {
		return fmt.Errorf("proto decode (offset %d): %w", r.Offset, err)
	}
	marshaler := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: true}
	value, err := marshaler.Marshal(msg.Interface())
	if err != nil {
		return err
	}
	rec := jsonRecord{
		Topic:     r.Topic,
		Partition: r.Partition,
		Offset:    r.Offset,
		Timestamp: r.Timestamp.UnixMilli(),
		Key:       string(r.Key),
		Value:     value,
	}
	if f.opts.Headers && len(r.Headers) > 0 {
		rec.Headers = make(map[string]string, len(r.Headers))
		for _, h := range r.Headers {
			rec.Headers[h.Key] = string(h.Value)
		}
	}
	return writeJSON(w, rec, f.opts.Pretty)
}
