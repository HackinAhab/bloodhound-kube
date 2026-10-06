package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type AsyncWriter struct {
	file   *os.File
	writer *bufio.Writer
	logger *Logger
}

func NewAsyncWriter(outputPath, filename string, log *Logger) (*AsyncWriter, error) {
	return newAsyncWriter(outputPath, filename, log, false)
}

func NewAsyncWriterAppend(outputPath, filename string, log *Logger) (*AsyncWriter, error) {
	return newAsyncWriter(outputPath, filename, log, true)
}

func newAsyncWriter(outputPath, filename string, log *Logger, appendMode bool) (*AsyncWriter, error) {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputPath, filename)

	var file *os.File
	var err error

	if appendMode {
		file, err = os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0644)
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
		// Persist the new file and its directory entry even when the checkpoint
		// lives in another directory and collection stops before its first job.
		if err := file.Sync(); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to persist output file: %w", err)
		}
		if err := SyncDirectory(outputPath); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to persist output directory: %w", err)
		}
	}

	writer := bufio.NewWriter(file)

	return &AsyncWriter{
		file:   file,
		writer: writer,
		logger: log,
	}, nil
}

func (w *AsyncWriter) WriteJSONLBatch(data []any) error {
	encoder := json.NewEncoder(w.writer)
	for _, item := range data {
		if err := encoder.Encode(item); err != nil {
			return fmt.Errorf("failed to encode JSONL item: %w", err)
		}
	}

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush JSONL batch: %w", err)
	}

	return nil
}

func (w *AsyncWriter) Flush() error {
	w.logger.Debug("Flushing writer buffer")
	return w.writer.Flush()
}

// Commit makes output durable before its byte position is checkpointed.
func (w *AsyncWriter) Commit() (int64, error) {
	if err := w.writer.Flush(); err != nil {
		return 0, err
	}
	if err := w.file.Sync(); err != nil {
		return 0, err
	}
	info, err := w.file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// Restore discards output written after the last committed checkpoint.
func (w *AsyncWriter) Restore(offset int64) error {
	info, err := w.file.Stat()
	if err != nil {
		return err
	}
	if offset < 0 || info.Size() < offset {
		return fmt.Errorf("output file is shorter than checkpoint position %d", offset)
	}
	if err := w.file.Truncate(offset); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *AsyncWriter) Close() error {
	w.logger.Debug("Closing async writer")

	var flushErr error
	if err := w.writer.Flush(); err != nil {
		flushErr = fmt.Errorf("flush: %w", err)
	}

	if err := w.file.Close(); err != nil {
		return errors.Join(flushErr, fmt.Errorf("close: %w", err))
	}

	return flushErr
}
