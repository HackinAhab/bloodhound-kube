package collector

import (
	"bloodhound-kube/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Checkpoint struct {
	Version       string             `json:"version"`
	Timestamp     string             `json:"timestamp"`
	Cluster       ClusterInfo        `json:"cluster"`
	CollectionID  string             `json:"collection_id"`
	OutputFile    string             `json:"output_file"`
	CompletedJobs []CompletedJob     `json:"completed_jobs"`
	FailedJobs    []FailedJob        `json:"failed_jobs"`
	TotalJobs     int                `json:"total_jobs"`
	JobsRemaining int                `json:"jobs_remaining"`
	Settings      json.RawMessage    `json:"settings,omitempty"`
	Pipeline      json.RawMessage    `json:"pipeline,omitempty"`
	Targets       []CollectionTarget `json:"targets"`
	Namespaces    []string           `json:"namespaces"`
	OutputOffset  int64              `json:"output_offset"`
	APIServer     string             `json:"api_server"`
}

type ClusterInfo struct {
	Type     string `json:"type"`
	Version  string `json:"version,omitempty"`
	Platform string `json:"platform"`
}

type CompletedJob struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace"`
	Count     int    `json:"count"`
	Duration  string `json:"duration"`
	Timestamp string `json:"timestamp"`
}

type FailedJob struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

func NewCheckpoint(collectionID, outputFile string, clusterType utils.ClusterType, clusterInfo *utils.ClusterInfo, totalJobs int) *Checkpoint {
	cluster := ClusterInfo{
		Type:     string(clusterType),
		Platform: clusterInfo.Platform,
	}

	if clusterInfo.Version != nil {
		cluster.Version = clusterInfo.Version.GitVersion
	}

	return &Checkpoint{
		Version:       "2.0",
		Timestamp:     time.Now().Format(time.RFC3339),
		Cluster:       cluster,
		CollectionID:  collectionID,
		OutputFile:    outputFile,
		CompletedJobs: make([]CompletedJob, 0),
		FailedJobs:    make([]FailedJob, 0),
		TotalJobs:     totalJobs,
		JobsRemaining: totalJobs,
	}
}

func (c *Checkpoint) AddCompletedJob(jobType, namespace string, count int, duration time.Duration) {
	job := CompletedJob{
		Type:      jobType,
		Namespace: namespace,
		Count:     count,
		Duration:  duration.String(),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	c.CompletedJobs = append(c.CompletedJobs, job)
	c.JobsRemaining--
	if c.JobsRemaining < 0 {
		c.JobsRemaining = 0
	}
}

func (c *Checkpoint) AddFailedJob(jobType, namespace, error string) {
	c.FailedJobs = append(c.FailedJobs, FailedJob{
		Type:      jobType,
		Namespace: namespace,
		Error:     error,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func (c *Checkpoint) IsJobCompleted(jobType, namespace string) bool {
	for _, job := range c.CompletedJobs {
		if job.Type == jobType && job.Namespace == namespace {
			return true
		}
	}
	return false
}

func (c *Checkpoint) GetProgress() (completed, total int, percentage float64) {
	completed = len(c.CompletedJobs)
	total = c.TotalJobs
	if total > 0 {
		percentage = float64(completed) / float64(total) * 100
	}
	return completed, total, percentage
}

func (c *Checkpoint) Save(checkpointFile string) error {
	c.Timestamp = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	dir := filepath.Dir(checkpointFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	tempFile := checkpointFile + ".tmp"
	f, err := os.OpenFile(tempFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to write checkpoint temp file: %w", err)
	}
	_, writeErr := f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to persist checkpoint: %w", errors.Join(writeErr, closeErr))
	}

	if err := os.Rename(tempFile, checkpointFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to move checkpoint file: %w", err)
	}

	return utils.SyncDirectory(dir)
}

// ValidateResume rejects checkpoints that cannot reproduce the original run.
func (c *Checkpoint) ValidateResume() error {
	if c.Version != "2.0" {
		return fmt.Errorf("checkpoint version %q cannot restore the original collection settings; start a new collection to create a version 2.0 checkpoint", c.Version)
	}
	if len(c.Settings) == 0 || len(c.Targets) == 0 || !filepath.IsAbs(c.OutputFile) || c.OutputOffset < 0 || c.APIServer == "" {
		return fmt.Errorf("checkpoint is missing a valid saved plan, settings, cluster identity or output position")
	}
	total := 0
	namespacesSeen := make(map[string]bool)
	for _, namespace := range c.Namespaces {
		if namespace == "" || namespacesSeen[namespace] {
			return fmt.Errorf("checkpoint contains invalid namespaces")
		}
		namespacesSeen[namespace] = true
	}
	jobs := make(map[string]bool)
	for _, target := range c.Targets {
		if target.Name == "" || target.Version == "" || target.Resource == "" || (target.FetchMode != FetchModeFull && target.FetchMode != FetchModeMetadata) {
			return fmt.Errorf("checkpoint contains an invalid resource target")
		}
		namespaces := c.Namespaces
		if target.ClusterScoped {
			namespaces = []string{""}
		}
		for _, ns := range namespaces {
			key := target.Name + "\x00" + ns
			if jobs[key] {
				return fmt.Errorf("checkpoint contains duplicate jobs")
			}
			jobs[key] = true
			total++
		}
	}
	completed := make(map[string]bool)
	for _, job := range c.CompletedJobs {
		key := job.Type + "\x00" + job.Namespace
		if !jobs[key] || completed[key] {
			return fmt.Errorf("checkpoint contains invalid completed jobs")
		}
		completed[key] = true
	}
	if total != c.TotalJobs || total-len(completed) != c.JobsRemaining {
		return fmt.Errorf("checkpoint progress does not match its saved plan")
	}
	return nil
}

func LoadCheckpoint(checkpointFile string) (*Checkpoint, error) {
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

func DefaultCheckpointPath(outputDir, filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return filepath.Join(outputDir, "."+name+".checkpoint.json")
}

func RemoveCheckpoint(checkpointFile string) error {
	if err := os.Remove(checkpointFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove checkpoint file: %w", err)
	}
	return nil
}
