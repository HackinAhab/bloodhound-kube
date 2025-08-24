package writer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kube-bloodhound/internal/logger"
)

type AsyncWriter struct {
	file   *os.File
	writer *bufio.Writer
	logger *logger.Logger
}

func NewAsyncWriter(outputPath, filename string, log *logger.Logger) (*AsyncWriter, error) {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputPath, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	writer := bufio.NewWriter(file)

	log.Info("Created output file", "path", filePath)

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

func GenerateFilename(namespace string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("kube-bloodhound-%s-%s.json", namespace, timestamp)
}
