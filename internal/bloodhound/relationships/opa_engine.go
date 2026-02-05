package relationships

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
)

// OPAEngine implements relationship detection using OPA/Rego policies
type OPAEngine struct {
	preparedQuery *rego.PreparedEvalQuery
	nodeQuery     *rego.PreparedEvalQuery // For node creation queries
	compiler      *ast.Compiler
	policyDir     string
	nodePolicyDir string // Directory for node creation policies

	// Performance settings
	chunkSize    int
	parallelEval bool

	// Caching
	cache   map[string][]BloodHoundEdge
	cacheMu sync.RWMutex
}

// NewOPAEngine creates a new OPA-based relationship engine
func NewOPAEngine(policyDir string) (*OPAEngine, error) {
	engine := &OPAEngine{
		policyDir:    policyDir,
		chunkSize:    10000, // Process 10K nodes per chunk
		parallelEval: true,  // Enable parallel evaluation
		cache:        make(map[string][]BloodHoundEdge),
	}

	if err := engine.compilePolicies(); err != nil {
		return nil, fmt.Errorf("failed to compile policies: %w", err)
	}

	return engine, nil
}

// compilePolicies loads and compiles all Rego policies from the policy directory
func (e *OPAEngine) compilePolicies() error {
	ctx := context.Background()

	// Check if policy directory exists
	if _, err := os.Stat(e.policyDir); os.IsNotExist(err) {
		return fmt.Errorf("policy directory does not exist: %s", e.policyDir)
	}

	// Find all .rego files in policy directory
	policyFiles, err := filepath.Glob(filepath.Join(e.policyDir, "*.rego"))
	if err != nil {
		return fmt.Errorf("failed to find policy files: %w", err)
	}

	if len(policyFiles) == 0 {
		return fmt.Errorf("no policy files found in %s", e.policyDir)
	}

	// Load all policies
	query := rego.New(
		rego.Query("data.kubernetes.relationships"),
		rego.Load(policyFiles, nil),
	)

	// Prepare query for efficient evaluation
	prepared, err := query.PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}

	e.preparedQuery = &prepared
	return nil
}

// ApplyRules evaluates all Rego policies against the given nodes
func (e *OPAEngine) ApplyRules(nodes []BloodHoundNode) ([]BloodHoundEdge, error) {
	ctx := context.Background()

	// Prepare hierarchical input for better performance
	input := e.prepareHierarchicalInput(nodes)

	// Evaluate policies
	results, err := e.preparedQuery.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Extract edges from results
	edges := e.extractEdges(results)

	// Deduplicate and sort
	edges = DeduplicateEdges(edges)
	SortEdgesByKind(edges)

	return edges, nil
}

// prepareHierarchicalInput organizes nodes by namespace and type for efficient querying
func (e *OPAEngine) prepareHierarchicalInput(nodes []BloodHoundNode) map[string]any {
	input := map[string]any{
		"cluster_scoped": make(map[string][]interface{}),
		"namespaces":     make(map[string]map[string][]interface{}),
	}

	clusterScoped := input["cluster_scoped"].(map[string][]interface{})
	namespaces := input["namespaces"].(map[string]map[string][]interface{})

	for _, node := range nodes {
		// Convert to interface{} for OPA
		nodeData := map[string]interface{}{
			"id":         node.ID,
			"kinds":      node.Kinds,
			"properties": node.Properties,
		}

		resourceType, _ := node.Properties["resource_type"].(string)
		namespace, _ := node.Properties["namespace"].(string)

		if namespace == "" {
			// Cluster-scoped resource
			clusterScoped[resourceType] = append(clusterScoped[resourceType], nodeData)
		} else {
			// Namespace-scoped resource
			if namespaces[namespace] == nil {
				namespaces[namespace] = make(map[string][]interface{})
			}
			namespaces[namespace][resourceType] = append(namespaces[namespace][resourceType], nodeData)
		}
	}

	return input
}

// extractEdges converts OPA result set to BloodHoundEdge array
func (e *OPAEngine) extractEdges(results rego.ResultSet) []BloodHoundEdge {
	var edges []BloodHoundEdge

	if len(results) == 0 {
		return edges
	}

	// Navigate OPA result structure
	// Results are in: data.kubernetes.relationships.<category>.<rule_name>
	for _, result := range results {
		if len(result.Expressions) == 0 {
			continue
		}

		// Extract the relationship data
		value := result.Expressions[0].Value
		if value == nil {
			continue
		}

		relationships, ok := value.(map[string]any)
		if !ok {
			continue
		}

		// Iterate through categories (rbac, storage, networking, etc.)
		for _, categoryData := range relationships {
			categoryMap, ok := categoryData.(map[string]any)
			if !ok {
				continue
			}

			// Iterate through rules in this category
			for _, ruleResult := range categoryMap {
				edges = append(edges, e.convertRuleResult(ruleResult)...)
			}
		}
	}

	return edges
}

// convertRuleResult converts a single rule's result to BloodHoundEdge array
func (e *OPAEngine) convertRuleResult(ruleResult any) []BloodHoundEdge {
	var edges []BloodHoundEdge

	// Handle both single edge and array of edges
	switch v := ruleResult.(type) {
	case []any:
		for _, item := range v {
			if edge := e.convertToEdge(item); edge != nil {
				edges = append(edges, *edge)
			}
		}
	case map[string]any:
		if edge := e.convertToEdge(v); edge != nil {
			edges = append(edges, *edge)
		}
	}

	return edges
}

// convertToEdge converts OPA edge format to BloodHoundEdge
func (e *OPAEngine) convertToEdge(data any) *BloodHoundEdge {
	edgeMap, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	edge := &BloodHoundEdge{
		Properties: make(map[string]any),
	}

	// Extract start reference
	if start, ok := edgeMap["start"].(map[string]any); ok {
		if matchBy, ok := start["match_by"].(string); ok {
			edge.Start.MatchBy = matchBy
		}
		if value, ok := start["value"].(string); ok {
			edge.Start.Value = value
		}
		if kind, ok := start["kind"].(string); ok {
			edge.Start.Kind = kind
		}
	}

	// Extract end reference
	if end, ok := edgeMap["end"].(map[string]any); ok {
		if matchBy, ok := end["match_by"].(string); ok {
			edge.End.MatchBy = matchBy
		}
		if value, ok := end["value"].(string); ok {
			edge.End.Value = value
		}
		if kind, ok := end["kind"].(string); ok {
			edge.End.Kind = kind
		}
	}

	// Extract kind
	if kind, ok := edgeMap["kind"].(string); ok {
		edge.Kind = kind
	}

	// Extract properties
	if props, ok := edgeMap["properties"].(map[string]any); ok {
		maps.Copy(edge.Properties, props)
	}

	return edge
}

// SetChunkSize sets the chunk size for streaming mode
func (e *OPAEngine) SetChunkSize(size int) {
	e.chunkSize = size
}

// SetParallelEval enables or disables parallel evaluation
func (e *OPAEngine) SetParallelEval(enabled bool) {
	e.parallelEval = enabled
}

// SetNodePolicyDir sets the directory for node creation policies
func (e *OPAEngine) SetNodePolicyDir(dir string) error {
	e.nodePolicyDir = dir
	return e.compileNodePolicies()
}

// compileNodePolicies loads and compiles node creation policies
func (e *OPAEngine) compileNodePolicies() error {
	if e.nodePolicyDir == "" {
		return fmt.Errorf("node policy directory not set")
	}

	ctx := context.Background()

	// Check if node policy directory exists
	if _, err := os.Stat(e.nodePolicyDir); os.IsNotExist(err) {
		return fmt.Errorf("node policy directory does not exist: %s", e.nodePolicyDir)
	}

	// Find all .rego files in node policy directory (including subdirectories)
	var policyFiles []string
	err := filepath.Walk(e.nodePolicyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".rego" {
			policyFiles = append(policyFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find node policy files: %w", err)
	}

	if len(policyFiles) == 0 {
		return fmt.Errorf("no policy files found in %s", e.nodePolicyDir)
	}

	// Create query to get all nodes from all packages
	// Query pattern: data.nodes.* to get nodes from all node policy packages
	query := rego.New(
		rego.Query("data.nodes"),
		rego.Load(policyFiles, nil),
	)

	// Prepare query for efficient evaluation
	prepared, err := query.PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare node query: %w", err)
	}

	e.nodeQuery = &prepared
	return nil
}

// QueryNodes evaluates node creation policies against raw resources
// Returns a slice of BloodHoundNode objects created from the resources
func (e *OPAEngine) QueryNodes(resources []map[string]interface{}) ([]BloodHoundNode, error) {
	if e.nodeQuery == nil {
		return nil, fmt.Errorf("node policies not compiled - call SetNodePolicyDir first")
	}

	ctx := context.Background()

	// Prepare input for OPA
	input := map[string]interface{}{
		"resources": resources,
	}

	// Evaluate node policies
	results, err := e.nodeQuery.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return nil, fmt.Errorf("node policy evaluation failed: %w", err)
	}

	// Extract nodes from results
	nodes := e.extractNodes(results)

	return nodes, nil
}

// extractNodes converts OPA result set to BloodHoundNode array
func (e *OPAEngine) extractNodes(results rego.ResultSet) []BloodHoundNode {
	var nodes []BloodHoundNode

	if len(results) == 0 {
		return nodes
	}

	// Navigate OPA result structure
	// Results are in: data.nodes.<package>.nodes
	for _, result := range results {
		if len(result.Expressions) == 0 {
			continue
		}

		// Extract the node data
		value := result.Expressions[0].Value
		if value == nil {
			continue
		}

		nodePackages, ok := value.(map[string]any)
		if !ok {
			continue
		}

		// Iterate through packages (storage, rbac, workloads, networking, cluster)
		for packageName, packageData := range nodePackages {
			// Skip the helpers package (utility functions only)
			if packageName == "helpers" {
				continue
			}

			packageMap, ok := packageData.(map[string]any)
			if !ok {
				continue
			}

			// Extract nodes from this package
			if nodesData, ok := packageMap["nodes"]; ok {
				extractedNodes := e.convertNodesResult(nodesData)
				nodes = append(nodes, extractedNodes...)
			}
		}
	}

	return nodes
}

// convertNodesResult converts OPA nodes result to BloodHoundNode array
func (e *OPAEngine) convertNodesResult(nodesData any) []BloodHoundNode {
	var nodes []BloodHoundNode

	// Handle set of nodes (OPA returns sets as arrays)
	switch v := nodesData.(type) {
	case []any:
		for _, item := range v {
			if node := e.convertToNode(item); node != nil {
				nodes = append(nodes, *node)
			}
		}
	case map[string]any:
		// Single node
		if node := e.convertToNode(v); node != nil {
			nodes = append(nodes, *node)
		}
	}

	return nodes
}

// convertToNode converts OPA node format to BloodHoundNode
func (e *OPAEngine) convertToNode(data any) *BloodHoundNode {
	nodeMap, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	node := &BloodHoundNode{
		Properties: make(map[string]any),
	}

	// Extract ID
	if id, ok := nodeMap["id"].(string); ok {
		node.ID = id
	} else {
		return nil // ID is required
	}

	// Extract kinds
	if kinds, ok := nodeMap["kinds"].([]any); ok {
		node.Kinds = make([]string, 0, len(kinds))
		for _, k := range kinds {
			if kindStr, ok := k.(string); ok {
				node.Kinds = append(node.Kinds, kindStr)
			}
		}
	}

	// Extract properties
	if props, ok := nodeMap["properties"].(map[string]any); ok {
		maps.Copy(node.Properties, props)
	}

	return node
}
