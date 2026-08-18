package collector

import (
	"bloodhound-kube/internal/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Checkpoint struct {
	Version       string         `json:"version"`
	Timestamp     string         `json:"timestamp"`
	Cluster       ClusterInfo    `json:"cluster"`
	CollectionID  string         `json:"collection_id"`
	OutputFile    string         `json:"output_file"`
	CompletedJobs []CompletedJob `json:"completed_jobs"`
	FailedJobs    []FailedJob    `json:"failed_jobs"`
	TotalJobs     int            `json:"total_jobs"`
	JobsRemaining int            `json:"jobs_remaining"`
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
		Version:       "1.0",
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
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint temp file: %w", err)
	}

	if err := os.Rename(tempFile, checkpointFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to move checkpoint file: %w", err)
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
