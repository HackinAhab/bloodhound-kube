package cli

import (
	"fmt"
	"os"

	"bloodhound-kube/internal/parser"
	"bloodhound-kube/internal/utils"
)

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
