package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// collectNodes collects Kubernetes Nodes (cluster-scoped)
func collectNodes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting nodes")
	c.logger.Debug("Starting node collection")

	nodeList, err := c.clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list nodes", "error", err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	c.logger.Debug("Retrieved node list", "count", len(nodeList.Items))

	nodes := make([]any, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		var internalIP, externalIP string
		for _, address := range node.Status.Addresses {
			switch address.Type {
			case "InternalIP":
				internalIP = address.Address
			case "ExternalIP":
				externalIP = address.Address
			}
		}

		nodes = append(nodes, Node{
			CommonResourceMeta: CommonResourceMeta{
				Name:        node.Name,
				Labels:      node.Labels,
				Annotations: AnnotationsCleaner(node.Annotations),
				CreatedAt:   node.CreationTimestamp.Time,
			},
			Hostname:         node.Name, // Use name as hostname fallback
			InternalIP:       internalIP,
			ExternalIP:       externalIP,
			PodCIDR:          node.Spec.PodCIDR,
			KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			OSImage:          node.Status.NodeInfo.OSImage,
			KernelVersion:    node.Status.NodeInfo.KernelVersion,
			Architecture:     node.Status.NodeInfo.Architecture,
			OperatingSystem:  node.Status.NodeInfo.OperatingSystem,
			Unschedulable:    node.Spec.Unschedulable,
		})
	}

	c.logger.Info("Successfully collected nodes", "count", len(nodes))
	c.logger.Debug("Node collection completed", "processed", len(nodes))
	return nodes, nil
}

// collectCRDs collects Kubernetes CustomResourceDefinitions (cluster-scoped)
func collectCRDs(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting custom resource definitions")
	c.logger.Debug("Starting CRD collection")

	crdList, err := c.clients.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list CRDs", "error", err)
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	c.logger.Debug("Retrieved CRD list", "count", len(crdList.Items))

	crds := make([]any, 0, len(crdList.Items))
	for _, crd := range crdList.Items {
		var versions []CRDVersion
		for _, version := range crd.Spec.Versions {
			versions = append(versions, CRDVersion{
				Name:    version.Name,
				Served:  version.Served,
				Storage: version.Storage,
			})
		}

		// Use the first version as the primary version for compatibility
		var primaryVersion string
		if len(versions) > 0 {
			primaryVersion = versions[0].Name
		}

		crds = append(crds, CRD{
			CommonResourceMeta: CommonResourceMeta{
				Name:        crd.Name,
				Labels:      crd.Labels,
				Annotations: AnnotationsCleaner(crd.Annotations),
				CreatedAt:   crd.CreationTimestamp.Time,
			},
			Group:      crd.Spec.Group,
			Kind:       crd.Spec.Names.Kind,
			Version:    primaryVersion,
			Scope:      string(crd.Spec.Scope),
			Plural:     crd.Spec.Names.Plural,
			Singular:   crd.Spec.Names.Singular,
			ShortNames: crd.Spec.Names.ShortNames,
			Categories: crd.Spec.Names.Categories,
			Versions:   versions,
		})
	}

	c.logger.Info("Successfully collected CRDs", "count", len(crds))
	c.logger.Debug("CRD collection completed", "processed", len(crds))
	return crds, nil
}
