package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// JSONLHandler receives a trimmed JSON line and its 1-based line number.
type JSONLHandler func(line int, raw []byte) error

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
