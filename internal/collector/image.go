package collector

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Image struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	CreatedAt       string            `json:"created_at"`
	DockerImageRepo string            `json:"docker_image_repo"`
	Tags            []ImageTag        `json:"tags,omitempty"`
	LookupPolicy    string            `json:"lookup_policy,omitempty"`
	ImportPolicy    bool              `json:"import_policy"`
	ReferencePolicy string            `json:"reference_policy,omitempty"`
}

type ImageTag struct {
	Name            string                `json:"name"`
	DockerImageRef  string                `json:"docker_image_ref,omitempty"`
	Generation      int64                 `json:"generation,omitempty"`
	ImportPolicy    bool                  `json:"import_policy"`
	ReferencePolicy string                `json:"reference_policy,omitempty"`
	From            *ImageTagImportSource `json:"from,omitempty"`
	Conditions      []ImageTagCondition   `json:"conditions,omitempty"`
}

type ImageTagImportSource struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type ImageTagCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

func (c *Collector) CollectImages(ctx context.Context, namespace string) ([]Image, error) {
	c.logger.Info("Collecting OpenShift images", "namespace", namespace)

	if !c.IsOpenShift() {
		c.logger.Debug("Not an OpenShift cluster, skipping images collection")
		return []Image{}, nil
	}

	imageStreamGVR := schema.GroupVersionResource{
		Group:    "image.openshift.io",
		Version:  "v1",
		Resource: "imagestreams",
	}

	dynamicClient := c.clients.Kubernetes.Discovery().RESTClient()
	result := dynamicClient.Get().
		AbsPath("/apis", imageStreamGVR.Group, imageStreamGVR.Version, "namespaces", namespace, imageStreamGVR.Resource).
		Do(ctx)

	rawData, err := result.Raw()
	if err != nil {
		c.logger.Debug("OpenShift images not available", "namespace", namespace, "error", err)
		return []Image{}, nil
	}

	imageStreamList := &unstructured.UnstructuredList{}
	if err := imageStreamList.UnmarshalJSON(rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image stream list: %w", err)
	}

	images := make([]Image, 0, len(imageStreamList.Items))
	for _, item := range imageStreamList.Items {
		image := Image{
			Name:        item.GetName(),
			Namespace:   item.GetNamespace(),
			Labels:      item.GetLabels(),
			Annotations: AnnotationsCleaner(item.GetAnnotations()),
		}

		if creationTime := item.GetCreationTimestamp(); !creationTime.IsZero() {
			image.CreatedAt = creationTime.Format("2006-01-02T15:04:05Z")
		}

		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			if lookupPolicy, found, _ := unstructured.NestedMap(spec, "lookupPolicy"); found {
				if local, found, _ := unstructured.NestedBool(lookupPolicy, "local"); found && local {
					image.LookupPolicy = "local"
				} else {
					image.LookupPolicy = "registry"
				}
			}

			if tagsRaw, found, _ := unstructured.NestedSlice(spec, "tags"); found {
				var tags []ImageTag
				for _, tagRaw := range tagsRaw {
					if tagMap, ok := tagRaw.(map[string]any); ok {
						tag := ImageTag{}
						if name, found, _ := unstructured.NestedString(tagMap, "name"); found {
							tag.Name = name
						}
						if generation, found, _ := unstructured.NestedInt64(tagMap, "generation"); found {
							tag.Generation = generation
						}

						if importPolicy, found, _ := unstructured.NestedMap(tagMap, "importPolicy"); found {
							if scheduled, found, _ := unstructured.NestedBool(importPolicy, "scheduled"); found {
								tag.ImportPolicy = scheduled
							}
						}

						if referencePolicy, found, _ := unstructured.NestedMap(tagMap, "referencePolicy"); found {
							if refType, found, _ := unstructured.NestedString(referencePolicy, "type"); found {
								tag.ReferencePolicy = refType
							}
						}

						if from, found, _ := unstructured.NestedMap(tagMap, "from"); found {
							fromSource := &ImageTagImportSource{}
							if kind, found, _ := unstructured.NestedString(from, "kind"); found {
								fromSource.Kind = kind
							}
							if ns, found, _ := unstructured.NestedString(from, "namespace"); found {
								fromSource.Namespace = ns
							}
							if name, found, _ := unstructured.NestedString(from, "name"); found {
								fromSource.Name = name
							}
							tag.From = fromSource
						}

						tags = append(tags, tag)
					}
				}
				image.Tags = tags
			}
		}

		if status, found, _ := unstructured.NestedMap(item.Object, "status"); found {
			if dockerImageRepo, found, _ := unstructured.NestedString(status, "dockerImageRepository"); found {
				image.DockerImageRepo = dockerImageRepo
			}

			if statusTagsRaw, found, _ := unstructured.NestedSlice(status, "tags"); found {
				for _, statusTagRaw := range statusTagsRaw {
					if statusTagMap, ok := statusTagRaw.(map[string]any); ok {
						if tagName, found, _ := unstructured.NestedString(statusTagMap, "tag"); found {
							for i, tag := range image.Tags {
								if tag.Name == tagName {
									if itemsRaw, found, _ := unstructured.NestedSlice(statusTagMap, "items"); found && len(itemsRaw) > 0 {
										if firstItem, ok := itemsRaw[0].(map[string]any); ok {
											if dockerImageRef, found, _ := unstructured.NestedString(firstItem, "dockerImageReference"); found {
												image.Tags[i].DockerImageRef = dockerImageRef
											}
										}
									}
									break
								}
							}
						}
					}
				}
			}
		}

		image.ImportPolicy = containsScheduledTag(image.Tags)
		image.ReferencePolicy = getMostCommonReferencePolicy(image.Tags)

		images = append(images, image)
	}

	c.logger.Info("Successfully collected images", "namespace", namespace, "count", len(images))
	return images, nil
}

func containsScheduledTag(tags []ImageTag) bool {
	for _, tag := range tags {
		if tag.ImportPolicy {
			return true
		}
	}
	return false
}

func getMostCommonReferencePolicy(tags []ImageTag) string {
	if len(tags) == 0 {
		return "source"
	}

	policyCount := make(map[string]int)
	for _, tag := range tags {
		if tag.ReferencePolicy != "" {
			policyCount[tag.ReferencePolicy]++
		}
	}

	if len(policyCount) == 0 {
		return "source"
	}

	maxCount := 0
	mostCommon := "source"
	for policy, count := range policyCount {
		if count > maxCount {
			maxCount = count
			mostCommon = policy
		}
	}

	return mostCommon
}

type ImagesHandler struct {
	*BaseHandler
}

func NewImagesHandler() *ImagesHandler {
	return &ImagesHandler{
		BaseHandler: &BaseHandler{
			name:          "images",
			clusterScoped: false,
		},
	}
}

func (h *ImagesHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	images, err := c.CollectImages(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(images))

	for _, image := range images {
		batch = append(batch, Resource{
			Type:      "image",
			Namespace: namespace,
			Resource:  image,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
