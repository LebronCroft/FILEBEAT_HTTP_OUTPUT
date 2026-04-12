package http

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestJSONLinesEncoderAddRawNil(t *testing.T) {
	enc := newJSONLinesEncoder(nil)

	if err := enc.AddRaw(nil); err != nil {
		t.Fatalf("AddRaw(nil) returned error: %v", err)
	}

	if got := enc.buf.String(); got != "null\n" {
		t.Fatalf("unexpected encoded output: %q", got)
	}
}

func TestGzipLinesEncoderAddRawNil(t *testing.T) {
	enc, err := newGzipLinesEncoder(gzip.DefaultCompression, nil)
	if err != nil {
		t.Fatalf("newGzipLinesEncoder returned error: %v", err)
	}

	if err := enc.AddRaw(nil); err != nil {
		t.Fatalf("AddRaw(nil) returned error: %v", err)
	}

	reader := enc.Reader()
	compressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read compressed payload: %v", err)
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("failed to open gzip payload: %v", err)
	}
	defer gzReader.Close()

	decoded, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to decode gzip payload: %v", err)
	}

	if got := string(decoded); got != "null\n" {
		t.Fatalf("unexpected encoded output: %q", got)
	}
}
