package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CRD struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
	Group       string            `json:"group"`
	Version     string            `json:"version"`
	Kind        string            `json:"kind"`
	Scope       string            `json:"scope"`
	Plural      string            `json:"plural"`
	Singular    string            `json:"singular,omitempty"`
	ShortNames  []string          `json:"short_names,omitempty"`
	Categories  []string          `json:"categories,omitempty"`
	Versions    []CRDVersion      `json:"versions,omitempty"`
	Conditions  []CRDCondition    `json:"conditions,omitempty"`
}

type CRDVersion struct {
	Name    string `json:"name"`
	Served  bool   `json:"served"`
	Storage bool   `json:"storage"`
}

type CRDCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

func (c *Collector) CollectCRDs(ctx context.Context) ([]CRD, error) {
	c.logger.Info("Collecting Custom Resource Definitions")

	crdList, err := c.clients.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	crds := make([]CRD, 0, len(crdList.Items))
	for _, crd := range crdList.Items {
		var versions []CRDVersion
		for _, version := range crd.Spec.Versions {
			versions = append(versions, CRDVersion{
				Name:    version.Name,
				Served:  version.Served,
				Storage: version.Storage,
			})
		}

		var conditions []CRDCondition
		for _, condition := range crd.Status.Conditions {
			conditions = append(conditions, CRDCondition{
				Type:    string(condition.Type),
				Status:  string(condition.Status),
				Reason:  condition.Reason,
				Message: condition.Message,
			})
		}

		preferredVersion := crd.Spec.Versions[0].Name
		for _, version := range crd.Spec.Versions {
			if version.Storage {
				preferredVersion = version.Name
				break
			}
		}

		var shortNames []string
		if crd.Spec.Names.ShortNames != nil {
			shortNames = crd.Spec.Names.ShortNames
		}

		var categories []string
		if crd.Spec.Names.Categories != nil {
			categories = crd.Spec.Names.Categories
		}

		crds = append(crds, CRD{
			Name:        crd.Name,
			Labels:      crd.Labels,
			Annotations: crd.Annotations,
			CreatedAt:   crd.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			Group:       crd.Spec.Group,
			Version:     preferredVersion,
			Kind:        crd.Spec.Names.Kind,
			Scope:       string(crd.Spec.Scope),
			Plural:      crd.Spec.Names.Plural,
			Singular:    crd.Spec.Names.Singular,
			ShortNames:  shortNames,
			Categories:  categories,
			Versions:    versions,
			Conditions:  conditions,
		})
	}

	c.logger.Info("Successfully collected CRDs", "count", len(crds))
	return crds, nil
}

type CRDHandler struct {
	*BaseHandler
}

func NewCRDHandler() *CRDHandler {
	return &CRDHandler{
		BaseHandler: &BaseHandler{
			name:          "crds",
			clusterScoped: true,
		},
	}
}

func (h *CRDHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	crds, err := c.CollectCRDs(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(crds))

	for _, crd := range crds {
		batch = append(batch, Resource{
			Type:      "crd",
			Resource:  crd,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
