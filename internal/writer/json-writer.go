package writer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bloodhound-kube/internal/logger"
)

type AsyncWriter struct {
	file   *os.File
	writer *bufio.Writer
	logger *logger.Logger
}

func NewAsyncWriter(outputPath, filename string, log *logger.Logger) (*AsyncWriter, error) {
	return newAsyncWriter(outputPath, filename, log, false)
}

func NewAsyncWriterAppend(outputPath, filename string, log *logger.Logger) (*AsyncWriter, error) {
	return newAsyncWriter(outputPath, filename, log, true)
}

func newAsyncWriter(outputPath, filename string, log *logger.Logger, appendMode bool) (*AsyncWriter, error) {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputPath, filename)

	var file *os.File
	var err error

	if appendMode {
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open output file for append: %w", err)
		}
		log.Info("Opened output file for append", "path", filePath)
	} else {
		file, err = os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		log.Info("Created output file", "path", filePath)
	}

	writer := bufio.NewWriter(file)

	return &AsyncWriter{
		file:   file,
		writer: writer,
		logger: log,
	}, nil
}

func (w *AsyncWriter) WriteJSON(data any) error {
	w.logger.Debug("Writing JSON data to file")

	encoder := json.NewEncoder(w.writer)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func (w *AsyncWriter) WriteNDJSONBatch(data []any) error {
	w.logger.Debug("Writing NDJSON batch to file", "count", len(data))

	encoder := json.NewEncoder(w.writer)
	for _, item := range data {
		if err := encoder.Encode(item); err != nil {
			return fmt.Errorf("failed to encode NDJSON item: %w", err)
		}
	}

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush NDJSON batch: %w", err)
	}

	return nil
}

func (w *AsyncWriter) WriteNDJSON(data any) error {
	w.logger.Debug("Writing NDJSON data to file")

	encoder := json.NewEncoder(w.writer)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode NDJSON: %w", err)
	}

	return nil
}

func (w *AsyncWriter) Flush() error {
	w.logger.Debug("Flushing writer buffer")
	return w.writer.Flush()
}

func (w *AsyncWriter) Close() error {
	w.logger.Debug("Closing async writer")

	if err := w.writer.Flush(); err != nil {
		w.logger.Error("Failed to flush buffer", "error", err)
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

func GenerateNDJSONFilename(namespace string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("bloodhound-kube-%s-%s.ndjson", namespace, timestamp)
}
