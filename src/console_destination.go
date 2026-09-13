package gosync

import (
	"errors"
	"fmt"
	"io"
)

type ConsoleDestination struct {
	writer io.Writer
}

func newConsoleDestination(writer io.Writer) ConsoleDestination {
	return ConsoleDestination{writer: writer}
}

func (destination ConsoleDestination) Write(formattedRecord string) error {
	if destination.writer == nil {
		return errors.New("console writer is required")
	}

	writtenBytes, err := destination.writer.Write([]byte(formattedRecord))
	if err != nil {
		return fmt.Errorf("write console log: %w", err)
	}
	if writtenBytes != len(formattedRecord) {
		return io.ErrShortWrite
	}

	return nil
}
