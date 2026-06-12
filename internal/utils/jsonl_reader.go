package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

// JSONLHandler receives a trimmed JSON line and its 1-based line number.
type JSONLHandler func(line int, raw []byte) error

// ReadJSONLFile reads a JSONL file and invokes handler for each line.
func ReadJSONLFile(path string, handler JSONLHandler) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open JSONL file: %w", err)
	}
	defer file.Close()

	return ReadJSONL(file, handler)
}

// ReadJSONL reads JSONL data from a reader and invokes handler for each line.
func ReadJSONL(reader io.Reader, handler JSONLHandler) error {
	if handler == nil {
		return fmt.Errorf("JSONL handler is required")
	}

	buffered := bufio.NewReader(reader)
	lineNum := 0

	for {
		lineBytes, err := buffered.ReadBytes('\n')
		if err != nil && len(lineBytes) == 0 {
			if err == io.EOF {
				break
			}
			return err
		}

		lineNum++
		trimmed := bytes.TrimSpace(lineBytes)
		if len(trimmed) == 0 {
			if err == io.EOF {
				break
			}
			continue
		}

		if err := handler(lineNum, trimmed); err != nil {
			return err
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}
