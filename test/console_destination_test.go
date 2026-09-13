package gosync_test

import (
	"bytes"
	"errors"
	"testing"
)

func TestConsoleDestinationWritesTheCompleteFormattedRecord(t *testing.T) {
	var output bytes.Buffer
	destination := newConsoleDestination(&output)
	record := "timestamp=2026-09-13T14:15:16Z severity=INFO event=change_synchronized message=\"done\"\n"

	if err := destination.Write(record); err != nil {
		t.Fatalf("write() error = %v, want nil", err)
	}
	if output.String() != record {
		t.Fatalf("console output = %q, want %q", output.String(), record)
	}
}

func TestConsoleDestinationReturnsWriterFailure(t *testing.T) {
	expectedError := errors.New("console unavailable")
	destination := newConsoleDestination(failingWriter{err: expectedError})

	err := destination.Write("record\n")
	if !errors.Is(err, expectedError) {
		t.Fatalf("write() error = %v, want wrapped console failure", err)
	}
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}
