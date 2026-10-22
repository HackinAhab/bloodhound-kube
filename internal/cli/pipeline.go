package cli

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bloodhound-kube/internal/collector"
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
	ClusterConcurrency  int
	Out                 io.Writer `json:"-"`
	ParentCheckpoint    string    `json:"parent_checkpoint,omitempty"`
}

type PipelineResponse struct {
	JSONLPath      string
	ParsedPath     string
	NodeCount      int
	EdgeCount      int
	Duration       time.Duration
	CheckpointPath string
	Phase          string
}

type PipelineService struct{}

func (s PipelineService) Run(ctx context.Context, req PipelineRequest, log *utils.Logger) (PipelineResponse, error) {
	if req.Collect.Resume {
		kind, err := checkpointKind(req.Collect.CheckpointFile)
		if err != nil {
			return PipelineResponse{}, err
		}
		if kind == multiRunKind {
			return resumeMultiPipeline(ctx, req, log)
		}
		if req.ClustersConfigPath != "" {
			return PipelineResponse{}, fmt.Errorf("--clusters-config can only refresh credentials when resuming a parent run checkpoint")
		}
	}
	if req.ClustersConfigPath != "" {
		return runMultiPipeline(ctx, req, log)
	}
	return runSinglePipeline(ctx, req, log)
}

func runSinglePipeline(ctx context.Context, req PipelineRequest, log *utils.Logger) (PipelineResponse, error) {
	start := time.Now()
	if req.Collect.Resume {
		var err error
		req, err = restorePipelineRequest(req)
		if err != nil {
			return PipelineResponse{}, err
		}
	}
	out := req.Out
	if out == nil {
		out = os.Stdout
	}
	// Store output behavior separately from the resolved collection settings.
	settings := req
	if settings.ParsedOutputPath != "" {
		path, err := filepath.Abs(settings.ParsedOutputPath)
		if err != nil {
			return PipelineResponse{}, err
		}
		if strings.HasSuffix(settings.ParsedOutputPath, "/") || settings.ParsedOutputPath == "." || settings.ParsedOutputPath == ".." {
			path += "/"
		}
		settings.ParsedOutputPath = path
	}
	settings.Collect = CollectRequest{}
	settings.ClustersConfigPath = ""
	var settingsErr error
	req.Collect.PipelineSettings, settingsErr = json.Marshal(settings)
	if settingsErr != nil {
		return PipelineResponse{}, settingsErr
	}

	var cp *collector.Checkpoint
	checkpointPath := req.Collect.CheckpointFile
	if req.Collect.Resume {
		resolved, err := resolveOutputAndCheckpoint(req.Collect)
		if err != nil {
			return PipelineResponse{}, err
		}
		cp = resolved.checkpoint
		if cp.JobsRemaining == 0 {
			writer, err := utils.NewAsyncWriterAppend(filepath.Dir(cp.OutputFile), filepath.Base(cp.OutputFile), log)
			if err != nil {
				return PipelineResponse{}, err
			}
			restoreErr := writer.Restore(cp.OutputOffset)
			if err := errors.Join(restoreErr, writer.Close()); err != nil {
				return PipelineResponse{}, err
			}
		}
		if cp.JobsRemaining == 0 && (cp.Phase == "" || cp.Phase == "collecting") {
			cp.Phase = "collected"
			if err := cp.Save(checkpointPath); err != nil {
				return PipelineResponse{}, err
			}
		}
	}
	var collectionErr error
	if cp == nil || cp.Phase == "" || cp.Phase == "collecting" {
		req.Collect.RetainCheckpoint = true
		collectResp, err := (CollectService{}).Run(ctx, req.Collect, out, log)
		collectionErr = err
		if err != nil {
			var partial *PartialCollectionError
			if !errors.As(err, &partial) || collectResp.OutputPath == "" {
				phase := "setup"
				if collectResp.OutputPath != "" {
					phase = "collecting"
				}
				return PipelineResponse{JSONLPath: collectResp.OutputPath, CheckpointPath: collectResp.CheckpointPath, Phase: phase, Duration: time.Since(start)}, err
			}
			log.Warn("Collection incomplete; parsing available resources", "jsonl_path", collectResp.OutputPath)
		}
		if collectResp.OutputPath == "" { // Discovery-list-only request.
			return PipelineResponse{Duration: time.Since(start)}, nil
		}
		checkpointPath = collectResp.CheckpointPath
		cp, err = collector.LoadCheckpoint(checkpointPath)
		if err != nil {
			return PipelineResponse{}, err
		}
	}
	response := func() PipelineResponse {
		parsed := ""
		if req.ParseEnabled {
			parsed = cp.Artifact
		}
		return PipelineResponse{JSONLPath: cp.OutputFile, ParsedPath: parsed, CheckpointPath: checkpointPath, Phase: cp.Phase, NodeCount: cp.NodeCount, EdgeCount: cp.EdgeCount, Duration: time.Since(start)}
	}
	if err := ctx.Err(); err != nil {
		return response(), err
	}
	parsedPath := resolveParsedOutputPath(cp.OutputFile, req.ParsedOutputPath)
	if req.ParseEnabled && (cp.Phase == "parsed" || cp.Phase == "complete") {
		if info, err := os.Stat(cp.Artifact); err != nil || !info.Mode().IsRegular() {
			return response(), fmt.Errorf("saved pipeline artifact %q is missing or invalid", cp.Artifact)
		}
	}
	if req.ParseEnabled && cp.Phase != "parsed" && cp.Phase != "complete" {
		parseResp, err := (ParseService{}).Run(ParseRequest{InputPath: cp.OutputFile, OutputPath: parsedPath, ClusterName: req.ClusterName, ParseUndefinedNodes: req.ParseUndefinedNodes}, out, log)
		if err != nil {
			return response(), errors.Join(collectionErr, fmt.Errorf("parse: %w", err))
		}
		cp.Artifact, cp.NodeCount, cp.EdgeCount = parsedPath, parseResp.NodeCount, parseResp.EdgeCount
		if collectionErr == nil {
			cp.Phase = "parsed"
			if err := cp.Save(checkpointPath); err != nil {
				return response(), err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return response(), err
	}
	if req.ParseEnabled && req.ZipOutput && cp.Phase != "complete" {
		zipPath, err := zipParsedOutput(parsedPath)
		if err != nil {
			return response(), errors.Join(collectionErr, fmt.Errorf("zip output: %w", err))
		}
		cp.Artifact = zipPath
	}
	if collectionErr != nil {
		return response(), collectionErr
	}
	if err := ctx.Err(); err != nil {
		return response(), err
	}
	cp.Phase = "complete"
	if err := cp.Save(checkpointPath); err != nil {
		return response(), err
	}
	if req.ParseEnabled && req.ZipOutput && cp.Artifact != parsedPath {
		if err := os.Remove(parsedPath); err != nil && !os.IsNotExist(err) {
			log.Warn("Could not remove intermediate parsed file", "path", parsedPath, "error", err)
		}
	}
	keepForParent := false
	if req.ParentCheckpoint != "" {
		_, err := os.Stat(req.ParentCheckpoint)
		keepForParent = !os.IsNotExist(err)
	}
	if !keepForParent {
		if err := collector.RemoveCheckpoint(checkpointPath); err != nil {
			return response(), err
		}
	}
	return response(), nil
}

func zipParsedOutput(jsonPath string) (string, error) {
	zipPath := parsedZipPath(jsonPath)

	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return "", fmt.Errorf("open json for zip: %w", err)
	}
	defer jsonFile.Close()

	zipFile, err := os.CreateTemp(filepath.Dir(zipPath), ".zip-*")
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}
	tempPath := zipFile.Name()
	defer os.Remove(tempPath)

	zw := zip.NewWriter(zipFile)
	entry, err := zw.Create(filepath.Base(jsonPath))
	if err != nil {
		zw.Close()
		zipFile.Close()
		return "", fmt.Errorf("create zip entry: %w", err)
	}
	if _, err = io.Copy(entry, jsonFile); err != nil {
		zw.Close()
		zipFile.Close()
		return "", fmt.Errorf("write zip entry: %w", err)
	}

	closeErr := zw.Close()
	if closeErr == nil {
		closeErr = zipFile.Sync()
	}
	fileErr := zipFile.Close()
	if closeErr != nil || fileErr != nil {
		return "", fmt.Errorf("finalize zip: %w", errors.Join(closeErr, fileErr))
	}
	if err := os.Rename(tempPath, zipPath); err != nil {
		return "", err
	}
	if err := utils.SyncDirectory(filepath.Dir(zipPath)); err != nil {
		return "", err
	}

	return zipPath, nil
}

func parsedZipPath(jsonPath string) string {
	if before, ok := strings.CutSuffix(jsonPath, ".json"); ok {
		return before + ".zip"
	}
	return jsonPath + ".zip"
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

func (s ParseService) Run(req ParseRequest, out io.Writer, log *utils.Logger) (ParseResponse, error) {
	if out == nil {
		out = os.Stdout
	}
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
		if err := utils.AtomicWriteFile(req.OutputPath, []byte(jsonData), 0644); err != nil {
			return ParseResponse{}, fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(out, "BloodHound Kubernetes data written to: %s\n", req.OutputPath)
		nodeCount, edgeCount := 0, 0
		if graph != nil {
			nodeCount = graph.GetNodeCount()
			edgeCount = graph.GetEdgeCount()
		}
		fmt.Fprintf(out, "Processed %d nodes and %d edges from cluster: %s\n", nodeCount, edgeCount, req.ClusterName)
		return ParseResponse{NodeCount: nodeCount, EdgeCount: edgeCount}, nil
	}

	fmt.Fprint(out, jsonData)
	nodeCount, edgeCount := 0, 0
	if graph != nil {
		nodeCount = graph.GetNodeCount()
		edgeCount = graph.GetEdgeCount()
	}
	return ParseResponse{NodeCount: nodeCount, EdgeCount: edgeCount}, nil
}
