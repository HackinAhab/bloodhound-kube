package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// collectDeployments collects Kubernetes Deployments from the specified namespace
func collectDeployments(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting deployments", "namespace", namespace)
	c.logger.Debug("Starting deployment collection", "namespace", namespace)

	deploymentList, err := c.clients.Kubernetes.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list deployments", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	c.logger.Debug("Retrieved deployment list", "namespace", namespace, "count", len(deploymentList.Items))

	deployments := make([]any, 0, len(deploymentList.Items))
	for _, deploy := range deploymentList.Items {
		var containerImages []string
		for _, container := range deploy.Spec.Template.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		var selector map[string]string
		if deploy.Spec.Selector != nil && deploy.Spec.Selector.MatchLabels != nil {
			selector = deploy.Spec.Selector.MatchLabels
		}

		deployment := Deployment{
			CommonResourceMeta: CommonResourceMeta{
				Name:        deploy.Name,
				Namespace:   deploy.Namespace,
				Labels:      deploy.Labels,
				Annotations: AnnotationsCleaner(deploy.Annotations),
				CreatedAt:   deploy.CreationTimestamp.Time,
			},
			Spec: DeploymentSpec{
				Selector:        selector,
				ContainerImages: containerImages,
			},
		}

		deployments = append(deployments, deployment)
	}

	c.logger.Info("Successfully collected deployments", "namespace", namespace, "count", len(deployments))
	c.logger.Debug("Deployment collection completed", "namespace", namespace, "processed", len(deployments))
	return deployments, nil
}

// collectDaemonSets collects Kubernetes DaemonSets from the specified namespace
func collectDaemonSets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting daemonsets", "namespace", namespace)
	c.logger.Debug("Starting daemonset collection", "namespace", namespace)

	daemonSetList, err := c.clients.Kubernetes.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list daemonsets", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}

	c.logger.Debug("Retrieved daemonset list", "namespace", namespace, "count", len(daemonSetList.Items))

	daemonSets := make([]any, 0, len(daemonSetList.Items))
	for _, ds := range daemonSetList.Items {
		var containerImages []string
		for _, container := range ds.Spec.Template.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		daemonSets = append(daemonSets, DaemonSet{
			CommonResourceMeta: CommonResourceMeta{
				Name:        ds.Name,
				Namespace:   ds.Namespace,
				Labels:      ds.Labels,
				Annotations: AnnotationsCleaner(ds.Annotations),
				CreatedAt:   ds.CreationTimestamp.Time,
			},
			DesiredNumber:   ds.Status.DesiredNumberScheduled,
			CurrentNumber:   ds.Status.CurrentNumberScheduled,
			ReadyNumber:     ds.Status.NumberReady,
			UpdatedNumber:   ds.Status.UpdatedNumberScheduled,
			AvailableNumber: ds.Status.NumberAvailable,
			ContainerImages: containerImages,
			Selector:        ds.Spec.Selector.MatchLabels,
		})
	}

	c.logger.Info("Successfully collected daemonsets", "namespace", namespace, "count", len(daemonSets))
	c.logger.Debug("Daemonset collection completed", "namespace", namespace, "processed", len(daemonSets))
	return daemonSets, nil
}

// collectStatefulSets collects Kubernetes StatefulSets from the specified namespace
func collectStatefulSets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting statefulsets", "namespace", namespace)
	c.logger.Debug("Starting statefulset collection", "namespace", namespace)

	statefulSetList, err := c.clients.Kubernetes.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list statefulsets", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}

	c.logger.Debug("Retrieved statefulset list", "namespace", namespace, "count", len(statefulSetList.Items))

	statefulSets := make([]any, 0, len(statefulSetList.Items))
	for _, sts := range statefulSetList.Items {
		var containerImages []string
		for _, container := range sts.Spec.Template.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		var volumeClaimTemplateNames []string
		for _, vct := range sts.Spec.VolumeClaimTemplates {
			volumeClaimTemplateNames = append(volumeClaimTemplateNames, vct.Name)
		}

		var selector map[string]string
		if sts.Spec.Selector != nil && sts.Spec.Selector.MatchLabels != nil {
			selector = sts.Spec.Selector.MatchLabels
		}

		statefulSets = append(statefulSets, StatefulSet{
			CommonResourceMeta: CommonResourceMeta{
				Name:        sts.Name,
				Namespace:   sts.Namespace,
				Labels:      sts.Labels,
				Annotations: AnnotationsCleaner(sts.Annotations),
				CreatedAt:   sts.CreationTimestamp.Time,
			},
			ServiceName:              sts.Spec.ServiceName,
			ContainerImages:          containerImages,
			Selector:                 selector,
			VolumeClaimTemplateNames: volumeClaimTemplateNames,
		})
	}

	c.logger.Info("Successfully collected statefulsets", "namespace", namespace, "count", len(statefulSets))
	c.logger.Debug("Statefulset collection completed", "namespace", namespace, "processed", len(statefulSets))
	return statefulSets, nil
}

// collectPods collects Kubernetes Pods from the specified namespace
func collectPods(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting pods", "namespace", namespace)
	c.logger.Debug("Starting pod collection", "namespace", namespace)

	podList, err := c.clients.Kubernetes.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list pods", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	c.logger.Debug("Retrieved pod list", "namespace", namespace, "count", len(podList.Items))

	pods := make([]any, 0, len(podList.Items))
	for _, pod := range podList.Items {
		var containerImages []string
		var containers []Container

		var allCapAdd []string
		var allCapDrop []string

		// Note: Pod-level resource limits are rarely used in Kubernetes
		// Most resource limits are defined at the container level
		// Pods do not need an aggregate of container limits - if pod-level limits
		// are not explicitly set in the pod spec, they should be null
		var podResourceLimits *ResourceLimits

		for _, container := range pod.Spec.Containers {
			containerImages = append(containerImages, container.Image)

			var containerSecurityContext *SecurityContext
			if container.SecurityContext != nil {
				containerSecurityContext = &SecurityContext{}

				if container.SecurityContext.AllowPrivilegeEscalation != nil {
					containerSecurityContext.AllowPrivEsc = container.SecurityContext.AllowPrivilegeEscalation
				}
				if container.SecurityContext.RunAsUser != nil {
					containerSecurityContext.RunAsUser = container.SecurityContext.RunAsUser
				}
				if container.SecurityContext.RunAsNonRoot != nil {
					containerSecurityContext.RunAsNonRoot = container.SecurityContext.RunAsNonRoot
				}
				if container.SecurityContext.RunAsGroup != nil {
					containerSecurityContext.RunAsGroup = container.SecurityContext.RunAsGroup
				}

				if container.SecurityContext.Capabilities != nil {
					var containerCapAdd []string
					var containerCapDrop []string

					if container.SecurityContext.Capabilities.Add != nil {
						for _, cap := range container.SecurityContext.Capabilities.Add {
							capStr := string(cap)
							containerCapAdd = append(containerCapAdd, capStr)
						}
					}
					if container.SecurityContext.Capabilities.Drop != nil {
						for _, cap := range container.SecurityContext.Capabilities.Drop {
							capStr := string(cap)
							containerCapDrop = append(containerCapDrop, capStr)
						}
					}

					if len(containerCapAdd) > 0 || len(containerCapDrop) > 0 {
						containerSecurityContext.LinuxCapabilities = &LinuxCapabilities{
							Add:  containerCapAdd,
							Drop: containerCapDrop,
						}
					}
				}
			}

			// Extract container resource limits
			var containerResourceLimits ResourceLimits
			if container.Resources.Requests != nil {
				if cpuReq := container.Resources.Requests.Cpu(); cpuReq != nil {
					containerResourceLimits.CpuReq = cpuReq.String()
				}
				if memReq := container.Resources.Requests.Memory(); memReq != nil {
					containerResourceLimits.MemReq = memReq.String()
				}
			}
			if container.Resources.Limits != nil {
				if cpuLimit := container.Resources.Limits.Cpu(); cpuLimit != nil {
					containerResourceLimits.CpuLimit = cpuLimit.String()
				}
				if memLimit := container.Resources.Limits.Memory(); memLimit != nil {
					containerResourceLimits.MemLimit = memLimit.String()
				}
			}

			containers = append(containers, Container{
				Name:            container.Name,
				Image:           container.Image,
				SecurityContext: containerSecurityContext,
				ResourceLimits:  containerResourceLimits,
			})
		}

		var securityContext *SecurityContext
		if pod.Spec.SecurityContext != nil || len(allCapAdd) > 0 || len(allCapDrop) > 0 {
			securityContext = &SecurityContext{}

			if pod.Spec.SecurityContext != nil {
				securityContext.RunAsUser = pod.Spec.SecurityContext.RunAsUser
				securityContext.RunAsGroup = pod.Spec.SecurityContext.RunAsGroup
				securityContext.RunAsNonRoot = pod.Spec.SecurityContext.RunAsNonRoot
				securityContext.FSGroup = pod.Spec.SecurityContext.FSGroup

				if pod.Spec.SecurityContext.SeccompProfile != nil {
					seccompProfile := &SeccompProfile{
						Type: string(pod.Spec.SecurityContext.SeccompProfile.Type),
					}
					if pod.Spec.SecurityContext.SeccompProfile.LocalhostProfile != nil {
						seccompProfile.LocalhostProfile = *pod.Spec.SecurityContext.SeccompProfile.LocalhostProfile
					}
					securityContext.SeccompProfile = seccompProfile
				}
			}
		}

		pods = append(pods, Pod{
			CommonResourceMeta: CommonResourceMeta{
				Name:        pod.Name,
				Namespace:   pod.Namespace,
				Labels:      pod.Labels,
				Annotations: AnnotationsCleaner(pod.Annotations),
				CreatedAt:   pod.CreationTimestamp.Time,
			},
			NodeName:        pod.Spec.NodeName,
			HostNetwork:     pod.Spec.HostNetwork,
			ContainerImages: containerImages,
			SecurityContext: securityContext,
			Containers:      containers,
			ServiceAccount:  pod.Spec.ServiceAccountName,
			ResourceLimits:  podResourceLimits,
		})
	}

	c.logger.Info("Successfully collected pods", "namespace", namespace, "count", len(pods))
	c.logger.Debug("Pod collection completed", "namespace", namespace, "processed", len(pods))
	return pods, nil
}
