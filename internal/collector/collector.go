package collector

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"kube-bloodhound/internal/k8s"
	"kube-bloodhound/internal/logger"
)

type Collector struct {
	client *kubernetes.Clientset
	logger *logger.Logger
}

func New(log *logger.Logger) (*Collector, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Collector{
		client: client,
		logger: log,
	}, nil
}
