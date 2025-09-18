package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Deployment struct {
	Name                    string            `json:"name"`
	Namespace               string            `json:"namespace"`
	Labels                  map[string]string `json:"labels,omitempty"`
	Annotations             map[string]string `json:"annotations,omitempty"`
	CreatedAt               string            `json:"created_at"`
	Replicas                int32             `json:"replicas"`
	ReadyReplicas           int32             `json:"ready_replicas"`
	AvailableReplicas       int32             `json:"available_replicas"`
	UnavailableReplicas     int32             `json:"unavailable_replicas"`
	UpdatedReplicas         int32             `json:"updated_replicas"`
	ObservedGeneration      int64             `json:"observed_generation"`
	StrategyType            string            `json:"strategy_type"`
	Selector                map[string]string `json:"selector,omitempty"`
	ContainerImages         []string          `json:"container_images,omitempty"`
	RevisionHistoryLimit    *int32            `json:"revision_history_limit,omitempty"`
	ProgressDeadlineSeconds *int32            `json:"progress_deadline_seconds,omitempty"`
}

func (c *Collector) CollectDeployments(ctx context.Context, namespace string) ([]Deployment, error) {
	c.logger.Info("Collecting deployments", "namespace", namespace)

	deploymentList, err := c.clients.Kubernetes.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]Deployment, 0, len(deploymentList.Items))
	for _, deploy := range deploymentList.Items {
		var containerImages []string
		for _, container := range deploy.Spec.Template.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		replicas := int32(0)
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}

		deployment := Deployment{
			Name:                    deploy.Name,
			Namespace:               deploy.Namespace,
			Labels:                  deploy.Labels,
			Annotations:             AnnotationsCleaner(deploy.Annotations),
			CreatedAt:               deploy.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			Replicas:                replicas,
			ReadyReplicas:           deploy.Status.ReadyReplicas,
			AvailableReplicas:       deploy.Status.AvailableReplicas,
			UnavailableReplicas:     deploy.Status.UnavailableReplicas,
			UpdatedReplicas:         deploy.Status.UpdatedReplicas,
			ObservedGeneration:      deploy.Status.ObservedGeneration,
			StrategyType:            string(deploy.Spec.Strategy.Type),
			ContainerImages:         containerImages,
			RevisionHistoryLimit:    deploy.Spec.RevisionHistoryLimit,
			ProgressDeadlineSeconds: deploy.Spec.ProgressDeadlineSeconds,
		}

		if deploy.Spec.Selector != nil && deploy.Spec.Selector.MatchLabels != nil {
			deployment.Selector = deploy.Spec.Selector.MatchLabels
		}

		deployments = append(deployments, deployment)
	}

	c.logger.Info("Successfully collected deployments", "count", len(deployments))
	return deployments, nil
}

type DeploymentsHandler struct {
	*BaseHandler
}

func NewDeploymentsHandler() *DeploymentsHandler {
	return &DeploymentsHandler{
		BaseHandler: &BaseHandler{
			name:          "deployments",
			clusterScoped: false,
		},
	}
}

func (h *DeploymentsHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	deployments, err := c.CollectDeployments(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(deployments))

	for _, deployment := range deployments {
		batch = append(batch, Resource{
			Type:      "deployment",
			Namespace: namespace,
			Resource:  deployment,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
