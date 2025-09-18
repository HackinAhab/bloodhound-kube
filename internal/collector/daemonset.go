package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSet struct {
	Name                   string            `json:"name"`
	Namespace              string            `json:"namespace"`
	Labels                 map[string]string `json:"labels,omitempty"`
	Annotations            map[string]string `json:"annotations,omitempty"`
	CreatedAt              string            `json:"created_at"`
	DesiredNumberScheduled int32             `json:"desired_number_scheduled"`
	CurrentNumberScheduled int32             `json:"current_number_scheduled"`
	NumberReady            int32             `json:"number_ready"`
	NumberAvailable        int32             `json:"number_available"`
	NumberUnavailable      int32             `json:"number_unavailable"`
	UpdatedNumberScheduled int32             `json:"updated_number_scheduled"`
	NumberMisscheduled     int32             `json:"number_misscheduled"`
	ObservedGeneration     int64             `json:"observed_generation"`
	UpdateStrategyType     string            `json:"update_strategy_type"`
	Selector               map[string]string `json:"selector,omitempty"`
	ContainerImages        []string          `json:"container_images,omitempty"`
	RevisionHistoryLimit   *int32            `json:"revision_history_limit,omitempty"`
}

func (c *Collector) CollectDaemonSets(ctx context.Context, namespace string) ([]DaemonSet, error) {
	c.logger.Info("Collecting daemonsets", "namespace", namespace)

	daemonSetList, err := c.clients.Kubernetes.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}

	daemonSets := make([]DaemonSet, 0, len(daemonSetList.Items))
	for _, ds := range daemonSetList.Items {
		var containerImages []string
		for _, container := range ds.Spec.Template.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		daemonSet := DaemonSet{
			Name:                   ds.Name,
			Namespace:              ds.Namespace,
			Labels:                 ds.Labels,
			Annotations:            AnnotationsCleaner(ds.Annotations),
			CreatedAt:              ds.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
			CurrentNumberScheduled: ds.Status.CurrentNumberScheduled,
			NumberReady:            ds.Status.NumberReady,
			NumberAvailable:        ds.Status.NumberAvailable,
			NumberUnavailable:      ds.Status.NumberUnavailable,
			UpdatedNumberScheduled: ds.Status.UpdatedNumberScheduled,
			NumberMisscheduled:     ds.Status.NumberMisscheduled,
			ObservedGeneration:     ds.Status.ObservedGeneration,
			UpdateStrategyType:     string(ds.Spec.UpdateStrategy.Type),
			ContainerImages:        containerImages,
			RevisionHistoryLimit:   ds.Spec.RevisionHistoryLimit,
		}

		if ds.Spec.Selector != nil && ds.Spec.Selector.MatchLabels != nil {
			daemonSet.Selector = ds.Spec.Selector.MatchLabels
		}

		daemonSets = append(daemonSets, daemonSet)
	}

	c.logger.Info("Successfully collected daemonsets", "count", len(daemonSets))
	return daemonSets, nil
}

type DaemonSetsHandler struct {
	*BaseHandler
}

func NewDaemonSetsHandler() *DaemonSetsHandler {
	return &DaemonSetsHandler{
		BaseHandler: &BaseHandler{
			name:          "daemonsets",
			clusterScoped: false,
		},
	}
}

func (h *DaemonSetsHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	daemonSets, err := c.CollectDaemonSets(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(daemonSets))

	for _, daemonSet := range daemonSets {
		batch = append(batch, Resource{
			Type:      "daemonset",
			Namespace: namespace,
			Resource:  daemonSet,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
