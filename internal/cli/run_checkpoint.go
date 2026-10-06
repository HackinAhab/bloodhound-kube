package cli

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/multicluster"
	"bloodhound-kube/internal/utils"
	"k8s.io/client-go/tools/clientcmd"
)

const multiRunKind = "multi-cluster"

type multiRunCheckpoint struct {
	Kind            string       `json:"kind"`
	Version         string       `json:"version"`
	ID              string       `json:"id"`
	Timestamp       string       `json:"timestamp"`
	ConfigPath      string       `json:"config_path"`
	Concurrency     int          `json:"cluster_concurrency"`
	CredentialsPath string       `json:"credentials_config,omitempty"`
	Clusters        []runCluster `json:"clusters"`
}

type runCluster struct {
	Name            string           `json:"name"`
	Request         PipelineRequest  `json:"request"`
	TokenEnv        string           `json:"token_env,omitempty"`
	TokenFromConfig bool             `json:"token_from_config,omitempty"`
	State           string           `json:"state"`
	Phase           string           `json:"phase"`
	LastError       string           `json:"last_error,omitempty"`
	Result          PipelineResponse `json:"result"`
}

func checkpointKind(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read checkpoint: %w", err)
	}
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return "", fmt.Errorf("invalid checkpoint: %w", err)
	}
	if header.Kind != "" && header.Kind != multiRunKind {
		return "", fmt.Errorf("unknown checkpoint kind %q", header.Kind)
	}
	return header.Kind, nil
}

func (run *multiRunCheckpoint) save(path string) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return utils.AtomicWriteFile(path, data, 0600)
}

func loadMultiRun(path string) (*multiRunCheckpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var run multiRunCheckpoint
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	if run.Kind != multiRunKind || run.Version != "1.0" || run.ID == "" || run.Concurrency < 1 || len(run.Clusters) == 0 {
		return nil, fmt.Errorf("invalid or unsupported parent run checkpoint")
	}
	names := make(map[string]bool)
	parentPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{parentPath: true}
	for _, c := range run.Clusters {
		if c.Name == "" || names[c.Name] || c.Request.ClusterName != c.Name || c.Request.ParentCheckpoint != parentPath {
			return nil, fmt.Errorf("invalid cluster identity in parent checkpoint")
		}
		names[c.Name] = true
		switch c.State {
		case "queued", "active", "failed", "complete":
		default:
			return nil, fmt.Errorf("invalid cluster state %q", c.State)
		}
		switch c.Phase {
		case "setup", "collecting", "collected", "parsed", "complete":
		default:
			return nil, fmt.Errorf("invalid cluster phase %q", c.Phase)
		}
		if c.State == "complete" && c.Phase != "complete" {
			return nil, fmt.Errorf("completed cluster has unfinished pipeline")
		}
		for _, p := range clusterRunPaths(c.Request) {
			if !filepath.IsAbs(p) || paths[filepath.Clean(p)] {
				return nil, fmt.Errorf("parent checkpoint contains invalid or overlapping output paths")
			}
			paths[filepath.Clean(p)] = true
		}
	}
	return &run, nil
}

func clusterRunPaths(req PipelineRequest) []string {
	paths := []string{req.Collect.Output, req.Collect.CheckpointFile}
	if req.ParseEnabled {
		paths = append(paths, req.ParsedOutputPath)
		if req.ZipOutput {
			paths = append(paths, parsedZipPath(req.ParsedOutputPath))
		}
	}
	return paths
}

func absoluteReference(path string) (string, error) {
	return filepath.Abs(utils.ExpandTildeInPath(path))
}

var tokenEnvPattern = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)

func createMultiRun(req PipelineRequest) (*multiRunCheckpoint, string, error) {
	cfg, err := multicluster.LoadConfig(req.ClustersConfigPath)
	if err != nil {
		return nil, "", err
	}
	// Validate before expanding tokens: only unfinished clusters need credentials.
	if err := multicluster.Validate(cfg); err != nil {
		return nil, "", err
	}
	entries := multicluster.ApplyDefaults(cfg)
	run := &multiRunCheckpoint{Kind: multiRunKind, Version: "1.0", ID: rand.Text(), Timestamp: time.Now().Format("2006-01-02-150405")}
	run.ConfigPath, err = filepath.Abs(req.ClustersConfigPath)
	if err != nil {
		return nil, "", err
	}
	run.Concurrency = req.ClusterConcurrency
	if run.Concurrency <= 0 {
		run.Concurrency = cfg.Defaults.ClusterConcurrency
	}
	if run.Concurrency <= 0 {
		run.Concurrency = 1
	}
	dir := req.Collect.Output
	if dir == "" {
		dir = cfg.Defaults.OutputDir
	}
	if dir == "" {
		dir = "."
	}
	parentPath := req.Collect.CheckpointFile
	if parentPath == "" {
		parentPath = filepath.Join(dir, ".run-"+run.Timestamp+"-"+run.ID+".checkpoint.json")
	}
	parentPath, err = filepath.Abs(parentPath)
	if err != nil {
		return nil, "", err
	}
	paths := map[string]bool{parentPath: true}
	for _, entry := range entries {
		child := buildClusterPipelineRequest(entry, req, run.Timestamp+"-"+run.ID)
		child.Collect.Resume = false
		child.ParentCheckpoint = parentPath
		child.Collect.Output, err = filepath.Abs(child.Collect.Output)
		if err != nil {
			return nil, "", err
		}
		if child.Collect.CheckpointFile == "" {
			child.Collect.CheckpointFile = collector.DefaultCheckpointPath(filepath.Dir(child.Collect.Output), filepath.Base(child.Collect.Output))
		} else {
			child.Collect.CheckpointFile, err = filepath.Abs(child.Collect.CheckpointFile)
			if err != nil {
				return nil, "", err
			}
		}
		if child.ParseEnabled {
			child.ParsedOutputPath, err = filepath.Abs(resolveParsedOutputPath(child.Collect.Output, child.ParsedOutputPath))
			if err != nil {
				return nil, "", err
			}
		}
		if err := pinQueuedConnection(&child.Collect); err != nil {
			return nil, "", err
		}
		if child.Collect.DiscoveryAllowlist != "" {
			child.Collect.DiscoveryAllowlist, err = absoluteReference(child.Collect.DiscoveryAllowlist)
			if err != nil {
				return nil, "", err
			}
		}
		for _, path := range clusterRunPaths(child) {
			if paths[path] {
				return nil, "", fmt.Errorf("clusters have overlapping checkpoint/output paths: %s", path)
			}
			paths[path] = true
			if _, err := os.Stat(path); err == nil {
				return nil, "", fmt.Errorf("run artifact already exists: %s; resume its checkpoint instead", path)
			} else if !os.IsNotExist(err) {
				return nil, "", err
			}
		}
		c := runCluster{Name: entry.Name, Request: child, State: "queued", Phase: "setup"}
		if match := tokenEnvPattern.FindStringSubmatch(entry.Token); match != nil {
			c.TokenEnv = match[1]
		} else if entry.Token != "" {
			c.TokenFromConfig = true
		}
		c.Request.Collect.Token = ""
		run.Clusters = append(run.Clusters, c)
	}
	if _, err := os.Stat(parentPath); err == nil {
		return nil, "", fmt.Errorf("parent checkpoint already exists: %s", parentPath)
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	if err := run.save(parentPath); err != nil {
		return nil, "", err
	}
	return run, parentPath, nil
}

// Resolve local connection references before clusters wait in the queue. No
// authentication or Kubernetes calls are needed to pin an available context.
func pinQueuedConnection(req *CollectRequest) error {
	if req.Server != "" {
		req.ExpectedAPIServer = req.Server
		return nil
	}
	if req.Kubeconfig == "" {
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
				return nil
			}
		}
	}
	path, err := utils.KubeconfigPath(req.Kubeconfig)
	if err != nil {
		return err
	}
	req.Kubeconfig = path
	if raw, err := clientcmd.LoadFromFile(path); err == nil {
		req.Context, req.ExpectedAPIServer = utils.KubeconfigIdentity(raw, req.Context)
	}
	return nil
}

func resolveRunCredentials(c runCluster, configPath string, override bool) (CollectRequest, error) {
	collect := c.Request.Collect
	if override || c.TokenFromConfig {
		cfg, err := multicluster.LoadConfig(configPath)
		if err != nil {
			return collect, fmt.Errorf("credentials config: %w", err)
		}
		found := false
		for _, entry := range cfg.Clusters {
			if entry.Name != c.Name {
				continue
			}
			if found {
				return collect, fmt.Errorf("credentials config has duplicate cluster %q", c.Name)
			}
			found = true
			if entry.Server != "" && entry.Server != collect.Server {
				return collect, fmt.Errorf("credentials config changes saved API server for %q", c.Name)
			}
			if override && entry.Kubeconfig != "" {
				if collect.Server != "" {
					return collect, fmt.Errorf("credentials config cannot change authentication mode for cluster %q", c.Name)
				}
				collect.Kubeconfig, err = absoluteReference(entry.Kubeconfig)
				if err != nil {
					return collect, err
				}
				collect.Server = ""
			}
			if override && entry.Context != "" {
				collect.Context = entry.Context
			}
			collect.Token, err = multicluster.ExpandToken(entry.Name, entry.Token)
			if err != nil {
				return collect, err
			}
		}
		if !found {
			return collect, fmt.Errorf("credentials config has no cluster %q", c.Name)
		}
	} else if c.TokenEnv != "" {
		collect.Token = os.Getenv(c.TokenEnv)
		if collect.Token == "" {
			return collect, fmt.Errorf("cluster %q: env var ${%s} is not set or empty", c.Name, c.TokenEnv)
		}
	}
	return collect, nil
}
