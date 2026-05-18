package cli

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bloodhound-kube/internal/parser"
	"bloodhound-kube/internal/utils"
)

type PipelineRequest struct {
	Collect             CollectRequest
	ParseEnabled        bool
	ParsedOutputPath    string
	ClusterName         string
	ParseUndefinedNodes bool
	ClustersConfigPath  string
	ZipOutput           bool
}

type PipelineResponse struct {
	JSONLPath  string
	ParsedPath string
	NodeCount  int
	EdgeCount  int
	Duration   time.Duration
}

type PipelineService struct{}

func (s PipelineService) Run(ctx context.Context, req PipelineRequest, log utils.Logger) (PipelineResponse, error) {
	if req.ClustersConfigPath != "" {
		return runMultiPipeline(ctx, req, log)
	}
	return runSinglePipeline(ctx, req, log)
}

func runSinglePipeline(ctx context.Context, req PipelineRequest, log utils.Logger) (PipelineResponse, error) {
	start := time.Now()

	collectResp, err := CollectService{}.Run(ctx, req.Collect, log)
	if err != nil {
		return PipelineResponse{}, err
	}

	jsonlPath := collectResp.OutputPath
	if jsonlPath == "" || !req.ParseEnabled {
		return PipelineResponse{
			JSONLPath: jsonlPath,
			Duration:  time.Since(start),
		}, nil
	}

	parsedPath := resolveParsedOutputPath(jsonlPath, req.ParsedOutputPath)
	parseResp, err := ParseService{}.Run(ParseRequest{
		InputPath:           jsonlPath,
		OutputPath:          parsedPath,
		ClusterName:         req.ClusterName,
		ParseUndefinedNodes: req.ParseUndefinedNodes,
	}, log)
	if err != nil {
		log.Error("Parse failed; JSONL artifact is intact", "jsonl_path", jsonlPath)
		return PipelineResponse{JSONLPath: jsonlPath, Duration: time.Since(start)}, err
	}

	if req.ZipOutput {
		zipPath, err := zipParsedOutput(parsedPath)
		if err != nil {
			log.Error("Failed to zip parsed output", "path", parsedPath, "error", err)
			return PipelineResponse{JSONLPath: jsonlPath, Duration: time.Since(start)}, fmt.Errorf("zip output: %w", err)
		}
		parsedPath = zipPath
	}

	return PipelineResponse{
		JSONLPath:  jsonlPath,
		ParsedPath: parsedPath,
		NodeCount:  parseResp.NodeCount,
		EdgeCount:  parseResp.EdgeCount,
		Duration:   time.Since(start),
	}, nil
}

func zipParsedOutput(jsonPath string) (string, error) {
	zipPath := strings.TrimSuffix(jsonPath, ".json") + ".zip"
	if !strings.HasSuffix(jsonPath, ".json") {
		zipPath = jsonPath + ".zip"
	}

	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return "", fmt.Errorf("open json for zip: %w", err)
	}
	defer jsonFile.Close()

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}

	zw := zip.NewWriter(zipFile)
	entry, err := zw.Create(filepath.Base(jsonPath))
	if err != nil {
		zw.Close()
		zipFile.Close()
		os.Remove(zipPath)
		return "", fmt.Errorf("create zip entry: %w", err)
	}
	if _, err = io.Copy(entry, jsonFile); err != nil {
		zw.Close()
		zipFile.Close()
		os.Remove(zipPath)
		return "", fmt.Errorf("write zip entry: %w", err)
	}

	closeErr := zw.Close()
	fileErr := zipFile.Close()
	if closeErr != nil || fileErr != nil {
		os.Remove(zipPath)
		return "", fmt.Errorf("finalize zip: %w", errors.Join(closeErr, fileErr))
	}

	if err := os.Remove(jsonPath); err != nil {
		return "", fmt.Errorf("remove json after zip: %w", err)
	}

	return zipPath, nil
}

func deriveParsedOutputPath(jsonlPath string) string {
	if before, ok := strings.CutSuffix(jsonlPath, ".jsonl"); ok {
		return before + ".json"
	}
	return jsonlPath + ".json"
}

func resolveParsedOutputPath(jsonlPath, parsedOutputPath string) string {
	if parsedOutputPath == "" {
		return deriveParsedOutputPath(jsonlPath)
	}
	if strings.HasSuffix(parsedOutputPath, "/") || parsedOutputPath == "." || parsedOutputPath == ".." {
		return filepath.Join(parsedOutputPath, filepath.Base(deriveParsedOutputPath(jsonlPath)))
	}
	return parsedOutputPath
}

type ParseRequest struct {
	InputPath           string
	OutputPath          string
	ClusterName         string
	ParseUndefinedNodes bool
}

type ParseResponse struct {
	NodeCount int
	EdgeCount int
}

type ParseService struct{}

func (s ParseService) Run(req ParseRequest, log utils.Logger) (ParseResponse, error) {
	if req.InputPath == "" {
		return ParseResponse{}, fmt.Errorf("input file is required")
	}

	file, err := os.Open(req.InputPath)
	if err != nil {
		return ParseResponse{}, fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	graph, err := parser.ConvertToBloodHoundResultFromReader(file, req.ClusterName, req.ParseUndefinedNodes)
	if err != nil {
		return ParseResponse{}, err
	}

	jsonData, err := graph.ExportJSON(true)
	if err != nil {
		return ParseResponse{}, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if req.OutputPath != "" {
		if err := os.WriteFile(req.OutputPath, []byte(jsonData), 0644); err != nil {
			return ParseResponse{}, fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("BloodHound Kubernetes data written to: %s\n", req.OutputPath)
		nodeCount, edgeCount := 0, 0
		if graph != nil {
			nodeCount = graph.GetNodeCount()
			edgeCount = graph.GetEdgeCount()
		}
		fmt.Printf("Processed %d nodes and %d edges from cluster: %s\n", nodeCount, edgeCount, req.ClusterName)
		return ParseResponse{NodeCount: nodeCount, EdgeCount: edgeCount}, nil
	}

	fmt.Print(jsonData)
	nodeCount, edgeCount := 0, 0
	if graph != nil {
		nodeCount = graph.GetNodeCount()
		edgeCount = graph.GetEdgeCount()
	}
	return ParseResponse{NodeCount: nodeCount, EdgeCount: edgeCount}, nil
}
