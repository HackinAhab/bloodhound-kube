package relationships

import (
	"fmt"
	"log"
)

const (
	DefaultChunkSize  = 10000
	DefaultBufferSize = 100
)

// StreamProcessor handles chunked processing of large node sets
type StreamProcessor struct {
	engine    *OPAEngine
	chunkSize int
}

// NewStreamProcessor creates a new streaming processor
func NewStreamProcessor(engine *OPAEngine, chunkSize int) *StreamProcessor {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	return &StreamProcessor{
		engine:    engine,
		chunkSize: chunkSize,
	}
}

// ProcessStream processes nodes in chunks and returns edge stream
func (sp *StreamProcessor) ProcessStream(nodes []BloodHoundNode) (<-chan []BloodHoundEdge, <-chan error) {
	edgeChan := make(chan []BloodHoundEdge, DefaultBufferSize)
	errChan := make(chan error, 1)

	go func() {
		defer close(edgeChan)
		defer close(errChan)

		// Split nodes into chunks
		for i := 0; i < len(nodes); i += sp.chunkSize {
			end := i + sp.chunkSize
			if end > len(nodes) {
				end = len(nodes)
			}

			chunk := nodes[i:end]

			// Process chunk
			edges, err := sp.engine.ApplyRules(chunk)
			if err != nil {
				log.Printf("Failed to process chunk %d-%d: %v", i, end, err)
				errChan <- fmt.Errorf("chunk %d-%d failed: %w", i, end, err)
				continue
			}

			if len(edges) > 0 {
				edgeChan <- edges
			}
		}
	}()

	return edgeChan, errChan
}

// AggregateEdges collects all edges from the stream
func (sp *StreamProcessor) AggregateEdges(edgeChan <-chan []BloodHoundEdge, errChan <-chan error) ([]BloodHoundEdge, error) {
	var allEdges []BloodHoundEdge

	// Collect edges from channel
	for edges := range edgeChan {
		allEdges = append(allEdges, edges...)
	}

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			return allEdges, err
		}
	default:
	}

	// Deduplicate across chunks
	allEdges = DeduplicateEdges(allEdges)

	return allEdges, nil
}

// ProcessStreamingMode is a convenience method that handles the entire streaming workflow
func (sp *StreamProcessor) ProcessStreamingMode(nodes []BloodHoundNode) ([]BloodHoundEdge, error) {
	edgeChan, errChan := sp.ProcessStream(nodes)
	return sp.AggregateEdges(edgeChan, errChan)
}
