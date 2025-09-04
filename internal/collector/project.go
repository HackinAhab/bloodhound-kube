package collector

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Project struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
	Status      string            `json:"status"`
	Requester   string            `json:"requester,omitempty"`
}

func (c *Collector) CollectProjects(ctx context.Context) ([]Project, error) {
	c.logger.Info("Collecting OpenShift projects")

	if !c.IsOpenShift() {
		c.logger.Debug("Not an OpenShift cluster, skipping projects collection")
		return []Project{}, nil
	}

	projectGVR := schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projects",
	}

	dynamicClient := c.clients.Kubernetes.Discovery().RESTClient()
	result := dynamicClient.Get().
		AbsPath("/apis", projectGVR.Group, projectGVR.Version, projectGVR.Resource).
		Do(ctx)

	rawData, err := result.Raw()
	if err != nil {
		c.logger.Debug("OpenShift projects not available", "error", err)
		return []Project{}, nil
	}

	projectList := &unstructured.UnstructuredList{}
	if err := projectList.UnmarshalJSON(rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project list: %w", err)
	}

	projects := make([]Project, 0, len(projectList.Items))
	for _, item := range projectList.Items {
		project := Project{
			Name:        item.GetName(),
			Labels:      item.GetLabels(),
			Annotations: item.GetAnnotations(),
		}

		if creationTime := item.GetCreationTimestamp(); !creationTime.IsZero() {
			project.CreatedAt = creationTime.Format("2006-01-02T15:04:05Z")
		}

		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			if displayName, found, _ := unstructured.NestedString(spec, "displayName"); found {
				project.DisplayName = displayName
			}
			if description, found, _ := unstructured.NestedString(spec, "description"); found {
				project.Description = description
			}
		}

		if status, found, _ := unstructured.NestedMap(item.Object, "status"); found {
			if phase, found, _ := unstructured.NestedString(status, "phase"); found {
				project.Status = phase
			}
		}

		if project.Annotations != nil {
			if requester, exists := project.Annotations["openshift.io/requester"]; exists {
				project.Requester = requester
			}
		}

		projects = append(projects, project)
	}

	c.logger.Info("Successfully collected projects", "count", len(projects))
	return projects, nil
}

type ProjectsHandler struct {
	*BaseHandler
}

func NewProjectsHandler() *ProjectsHandler {
	return &ProjectsHandler{
		BaseHandler: &BaseHandler{
			name:          "projects",
			clusterScoped: true,
		},
	}
}

func (h *ProjectsHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	projects, err := c.CollectProjects(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(projects))

	for _, project := range projects {
		batch = append(batch, Resource{
			Type:      "project",
			Resource:  project,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}