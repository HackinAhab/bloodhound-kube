package collector

import (
	"context"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// This file contains all the actual collection logic for each resource type.
// Each function follows the CollectFunc signature and contains the specific
// Kubernetes API calls and data transformation logic.

// collectConfigMaps collects Kubernetes ConfigMaps from the specified namespace
func collectConfigMaps(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting configmaps", "namespace", namespace)

	configMapList, err := c.clients.Kubernetes.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	configMaps := make([]any, 0, len(configMapList.Items))
	for _, cm := range configMapList.Items {
		var dataKeys []string
		dataMap := make(map[string]string)
		for key, value := range cm.Data {
			dataKeys = append(dataKeys, key)
			dataMap[key] = value
		}

		var binaryDataKeys []string
		binaryDataMap := make(map[string][]byte)
		for key, value := range cm.BinaryData {
			binaryDataKeys = append(binaryDataKeys, key)
			binaryDataMap[key] = value
		}

		configMaps = append(configMaps, ConfigMap{
			CommonResourceMeta: CommonResourceMeta{
				Name:        cm.Name,
				Namespace:   cm.Namespace,
				Labels:      cm.Labels,
				Annotations: AnnotationsCleaner(cm.Annotations),
				CreatedAt:   cm.CreationTimestamp.Time,
			},
			DataKeys:       dataKeys,
			Data:           dataMap,
			BinaryDataKeys: binaryDataKeys,
			BinaryData:     binaryDataMap,
		})
	}

	c.logger.Info("Successfully collected configmaps", "count", len(configMaps))
	return configMaps, nil
}

// collectSecrets collects Kubernetes Secrets from the specified namespace
func collectSecrets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting secrets", "namespace", namespace)

	secretList, err := c.clients.Kubernetes.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	secrets := make([]any, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var dataKeys []string
		dataMap := make(map[string]string)
		for key, value := range secret.Data {
			dataKeys = append(dataKeys, key)
			dataMap[key] = string(value)
		}
		secrets = append(secrets, Secret{
			CommonResourceMeta: CommonResourceMeta{
				Name:        secret.Name,
				Namespace:   secret.Namespace,
				Labels:      secret.Labels,
				Annotations: AnnotationsCleaner(secret.Annotations),
				CreatedAt:   secret.CreationTimestamp.Time,
			},
			Type:     string(secret.Type),
			DataKeys: dataKeys,
			Data:     dataMap,
		})
	}

	c.logger.Info("Successfully collected secrets", "count", len(secrets))
	return secrets, nil
}

// collectDeployments collects Kubernetes Deployments from the specified namespace
func collectDeployments(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting deployments", "namespace", namespace)

	deploymentList, err := c.clients.Kubernetes.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]any, 0, len(deploymentList.Items))
	for _, deploy := range deploymentList.Items {
		var containerImages []string
		for _, container := range deploy.Spec.Template.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		replicas := int32(0)
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
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
				Replicas:                &replicas,
				Selector:                selector,
				StrategyType:            string(deploy.Spec.Strategy.Type),
				ContainerImages:         containerImages,
				RevisionHistoryLimit:    deploy.Spec.RevisionHistoryLimit,
				ProgressDeadlineSeconds: deploy.Spec.ProgressDeadlineSeconds,
			},
			Status: DeploymentStatus{
				Replicas:            replicas,
				ReadyReplicas:       deploy.Status.ReadyReplicas,
				AvailableReplicas:   deploy.Status.AvailableReplicas,
				UnavailableReplicas: deploy.Status.UnavailableReplicas,
				UpdatedReplicas:     deploy.Status.UpdatedReplicas,
				ObservedGeneration:  deploy.Status.ObservedGeneration,
			},
		}

		deployments = append(deployments, deployment)
	}

	c.logger.Info("Successfully collected deployments", "count", len(deployments))
	return deployments, nil
}

// collectNodes collects Kubernetes Nodes (cluster-scoped)
func collectNodes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting nodes")

	nodeList, err := c.clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

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

		var taints []NodeTaint
		for _, taint := range node.Spec.Taints {
			taints = append(taints, NodeTaint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: string(taint.Effect),
			})
		}

		nodes = append(nodes, Node{
			Name:             node.Name,
			Labels:           node.Labels,
			Annotations:      AnnotationsCleaner(node.Annotations),
			CreatedAt:        node.CreationTimestamp.Time,
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
			Capacity: NodeResources{
				CPU:              node.Status.Capacity.Cpu().String(),
				Memory:           node.Status.Capacity.Memory().String(),
				EphemeralStorage: node.Status.Capacity.StorageEphemeral().String(),
				Pods:             node.Status.Capacity.Pods().String(),
			},
			Allocatable: NodeResources{
				CPU:              node.Status.Allocatable.Cpu().String(),
				Memory:           node.Status.Allocatable.Memory().String(),
				EphemeralStorage: node.Status.Allocatable.StorageEphemeral().String(),
				Pods:             node.Status.Allocatable.Pods().String(),
			},
			Taints: taints,
		})
	}

	c.logger.Info("Successfully collected nodes", "count", len(nodes))
	return nodes, nil
}

// collectServices collects Kubernetes Services from the specified namespace
func collectServices(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting services", "namespace", namespace)

	serviceList, err := c.clients.Kubernetes.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services := make([]any, 0, len(serviceList.Items))
	for _, svc := range serviceList.Items {
		var ports []ServicePort
		for _, port := range svc.Spec.Ports {
			var targetPort string
			if port.TargetPort.Type == intstr.Int {
				targetPort = strconv.Itoa(int(port.TargetPort.IntVal))
			} else {
				targetPort = port.TargetPort.StrVal
			}

			ports = append(ports, ServicePort{
				Name:       port.Name,
				Protocol:   string(port.Protocol),
				Port:       port.Port,
				TargetPort: targetPort,
				NodePort:   port.NodePort,
			})
		}

		services = append(services, Service{
			CommonResourceMeta: CommonResourceMeta{
				Name:        svc.Name,
				Namespace:   svc.Namespace,
				Labels:      svc.Labels,
				Annotations: AnnotationsCleaner(svc.Annotations),
				CreatedAt:   svc.CreationTimestamp.Time,
			},
			Type:            string(svc.Spec.Type),
			ClusterIP:       svc.Spec.ClusterIP,
			ExternalIPs:     svc.Spec.ExternalIPs,
			LoadBalancerIP:  svc.Spec.LoadBalancerIP,
			Ports:           ports,
			Selector:        svc.Spec.Selector,
			SessionAffinity: string(svc.Spec.SessionAffinity),
			ExternalName:    svc.Spec.ExternalName,
		})
	}

	c.logger.Info("Successfully collected services", "count", len(services))
	return services, nil
}

// collectIngresses collects Kubernetes Ingresses from the specified namespace
func collectIngresses(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting ingresses", "namespace", namespace)

	ingressList, err := c.clients.Kubernetes.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	ingresses := make([]any, 0, len(ingressList.Items))
	for _, ing := range ingressList.Items {
		var hosts []string
		var paths []IngressPath
		var tls []IngressTLS

		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					var port string
					if path.Backend.Service != nil {
						if path.Backend.Service.Port.Number != 0 {
							port = strconv.Itoa(int(path.Backend.Service.Port.Number))
						} else {
							port = path.Backend.Service.Port.Name
						}
						paths = append(paths, IngressPath{
							Host:    rule.Host,
							Path:    path.Path,
							Service: path.Backend.Service.Name,
							Port:    port,
						})
					}
				}
			}
		}

		for _, tlsSpec := range ing.Spec.TLS {
			tls = append(tls, IngressTLS{
				Hosts:      tlsSpec.Hosts,
				SecretName: tlsSpec.SecretName,
			})
		}

		ingresses = append(ingresses, Ingress{
			CommonResourceMeta: CommonResourceMeta{
				Name:        ing.Name,
				Namespace:   ing.Namespace,
				Labels:      ing.Labels,
				Annotations: AnnotationsCleaner(ing.Annotations),
				CreatedAt:   ing.CreationTimestamp.Time,
			},
			Hosts: hosts,
			Paths: paths,
			TLS:   tls,
		})
	}

	c.logger.Info("Successfully collected ingresses", "count", len(ingresses))
	return ingresses, nil
}

// collectGateways collects Gateway API resources from the specified namespace
func collectGateways(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting gateways", "namespace", namespace)
	//TODO
	return []any{}, nil
}

// collectRBAC collects RBAC resources (cluster-scoped: Roles, ClusterRoles, RoleBindings, ClusterRoleBindings)
func collectRBAC(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting RBAC resources")

	var rbacResources []any

	// Collect Roles (namespaced)
	if namespace != "" {
		roleList, err := c.clients.Kubernetes.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, role := range roleList.Items {
				var rules []PolicyRule
				for _, rule := range role.Rules {
					rules = append(rules, PolicyRule{
						APIGroups:     rule.APIGroups,
						Resources:     rule.Resources,
						Verbs:         rule.Verbs,
						ResourceNames: rule.ResourceNames,
					})
				}

				rbacResources = append(rbacResources, RBACResource{
					CommonResourceMeta: CommonResourceMeta{
						Name:        role.Name,
						Namespace:   role.Namespace,
						Labels:      role.Labels,
						Annotations: AnnotationsCleaner(role.Annotations),
						CreatedAt:   role.CreationTimestamp.Time,
					},
					Kind:  "Role",
					Rules: rules,
				})
			}
		}

		// Collect RoleBindings (namespaced)
		roleBindingList, err := c.clients.Kubernetes.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, rb := range roleBindingList.Items {
				var subjects []RBACSubject
				for _, subject := range rb.Subjects {
					subjects = append(subjects, RBACSubject{
						Kind:      subject.Kind,
						Name:      subject.Name,
						Namespace: subject.Namespace,
					})
				}

				rbacResources = append(rbacResources, RBACResource{
					CommonResourceMeta: CommonResourceMeta{
						Name:        rb.Name,
						Namespace:   rb.Namespace,
						Labels:      rb.Labels,
						Annotations: AnnotationsCleaner(rb.Annotations),
						CreatedAt:   rb.CreationTimestamp.Time,
					},
					Kind:     "RoleBinding",
					Subjects: subjects,
					RoleRef: &RoleRef{
						Kind:     rb.RoleRef.Kind,
						Name:     rb.RoleRef.Name,
						APIGroup: rb.RoleRef.APIGroup,
					},
				})
			}
		}
	} else {
		// Collect cluster-scoped RBAC resources when namespace is empty

		// Collect ClusterRoles
		clusterRoleList, err := c.clients.Kubernetes.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, cr := range clusterRoleList.Items {
				var rules []PolicyRule
				for _, rule := range cr.Rules {
					rules = append(rules, PolicyRule{
						APIGroups:     rule.APIGroups,
						Resources:     rule.Resources,
						Verbs:         rule.Verbs,
						ResourceNames: rule.ResourceNames,
					})
				}

				rbacResources = append(rbacResources, RBACResource{
					CommonResourceMeta: CommonResourceMeta{
						Name:        cr.Name,
						Labels:      cr.Labels,
						Annotations: AnnotationsCleaner(cr.Annotations),
						CreatedAt:   cr.CreationTimestamp.Time,
					},
					Kind:  "ClusterRole",
					Rules: rules,
				})
			}
		}

		// Collect ClusterRoleBindings
		clusterRoleBindingList, err := c.clients.Kubernetes.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, crb := range clusterRoleBindingList.Items {
				var subjects []RBACSubject
				for _, subject := range crb.Subjects {
					subjects = append(subjects, RBACSubject{
						Kind:      subject.Kind,
						Name:      subject.Name,
						Namespace: subject.Namespace,
					})
				}

				rbacResources = append(rbacResources, RBACResource{
					CommonResourceMeta: CommonResourceMeta{
						Name:        crb.Name,
						Labels:      crb.Labels,
						Annotations: AnnotationsCleaner(crb.Annotations),
						CreatedAt:   crb.CreationTimestamp.Time,
					},
					Kind:     "ClusterRoleBinding",
					Subjects: subjects,
					RoleRef: &RoleRef{
						Kind:     crb.RoleRef.Kind,
						Name:     crb.RoleRef.Name,
						APIGroup: crb.RoleRef.APIGroup,
					},
				})
			}
		}
	}

	c.logger.Info("Successfully collected RBAC resources", "count", len(rbacResources))
	return rbacResources, nil
}

// collectNetworkPolicies collects Kubernetes NetworkPolicies from the specified namespace
func collectNetworkPolicies(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting network policies", "namespace", namespace)

	npList, err := c.clients.Kubernetes.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list network policies: %w", err)
	}

	networkPolicies := make([]any, 0, len(npList.Items))
	for _, np := range npList.Items {
		var policyTypes []string
		for _, pt := range np.Spec.PolicyTypes {
			policyTypes = append(policyTypes, string(pt))
		}

		var ingress []NetworkPolicyIngressRule
		for _, rule := range np.Spec.Ingress {
			var ports []NetworkPolicyPort
			for _, port := range rule.Ports {
				var protocol, portStr string
				if port.Protocol != nil {
					protocol = string(*port.Protocol)
				}
				if port.Port != nil {
					portStr = port.Port.String()
				}
				ports = append(ports, NetworkPolicyPort{
					Protocol: protocol,
					Port:     portStr,
				})
			}

			var from []NetworkPolicyPeer
			for _, peer := range rule.From {
				npPeer := NetworkPolicyPeer{}
				if peer.PodSelector != nil {
					npPeer.PodSelector = peer.PodSelector.MatchLabels
				}
				if peer.NamespaceSelector != nil {
					npPeer.NamespaceSelector = peer.NamespaceSelector.MatchLabels
				}
				from = append(from, npPeer)
			}

			ingress = append(ingress, NetworkPolicyIngressRule{
				Ports: ports,
				From:  from,
			})
		}

		var egress []NetworkPolicyEgressRule
		for _, rule := range np.Spec.Egress {
			var ports []NetworkPolicyPort
			for _, port := range rule.Ports {
				var protocol, portStr string
				if port.Protocol != nil {
					protocol = string(*port.Protocol)
				}
				if port.Port != nil {
					portStr = port.Port.String()
				}
				ports = append(ports, NetworkPolicyPort{
					Protocol: protocol,
					Port:     portStr,
				})
			}

			var to []NetworkPolicyPeer
			for _, peer := range rule.To {
				npPeer := NetworkPolicyPeer{}
				if peer.PodSelector != nil {
					npPeer.PodSelector = peer.PodSelector.MatchLabels
				}
				if peer.NamespaceSelector != nil {
					npPeer.NamespaceSelector = peer.NamespaceSelector.MatchLabels
				}
				to = append(to, npPeer)
			}

			egress = append(egress, NetworkPolicyEgressRule{
				Ports: ports,
				To:    to,
			})
		}

		networkPolicies = append(networkPolicies, NetworkPolicy{
			CommonResourceMeta: CommonResourceMeta{
				Name:        np.Name,
				Namespace:   np.Namespace,
				Labels:      np.Labels,
				Annotations: AnnotationsCleaner(np.Annotations),
				CreatedAt:   np.CreationTimestamp.Time,
			},
			PodSelector: np.Spec.PodSelector.MatchLabels,
			PolicyTypes: policyTypes,
			Ingress:     ingress,
			Egress:      egress,
		})
	}

	c.logger.Info("Successfully collected network policies", "count", len(networkPolicies))
	return networkPolicies, nil
}

// collectCRDs collects Kubernetes CustomResourceDefinitions (cluster-scoped)
func collectCRDs(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting custom resource definitions")

	crdList, err := c.clients.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

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
	return crds, nil
}

// collectDaemonSets collects Kubernetes DaemonSets from the specified namespace
func collectDaemonSets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting daemonsets", "namespace", namespace)

	daemonSetList, err := c.clients.Kubernetes.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}

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

	c.logger.Info("Successfully collected daemonsets", "count", len(daemonSets))
	return daemonSets, nil
}

// collectStatefulSets collects Kubernetes StatefulSets from the specified namespace
func collectStatefulSets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting statefulsets", "namespace", namespace)

	statefulSetList, err := c.clients.Kubernetes.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}

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

		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}

		var selector map[string]string
		if sts.Spec.Selector != nil && sts.Spec.Selector.MatchLabels != nil {
			selector = sts.Spec.Selector.MatchLabels
		}

		var partition *int32
		if sts.Spec.UpdateStrategy.RollingUpdate != nil && sts.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
			partition = sts.Spec.UpdateStrategy.RollingUpdate.Partition
		}

		statefulSets = append(statefulSets, StatefulSet{
			CommonResourceMeta: CommonResourceMeta{
				Name:        sts.Name,
				Namespace:   sts.Namespace,
				Labels:      sts.Labels,
				Annotations: AnnotationsCleaner(sts.Annotations),
				CreatedAt:   sts.CreationTimestamp.Time,
			},
			Replicas:                 replicas,
			ReadyReplicas:            sts.Status.ReadyReplicas,
			CurrentReplicas:          sts.Status.CurrentReplicas,
			UpdatedReplicas:          sts.Status.UpdatedReplicas,
			ObservedGeneration:       sts.Status.ObservedGeneration,
			ServiceName:              sts.Spec.ServiceName,
			PodManagementPolicy:      string(sts.Spec.PodManagementPolicy),
			UpdateStrategyType:       string(sts.Spec.UpdateStrategy.Type),
			Partition:                partition,
			ContainerImages:          containerImages,
			Selector:                 selector,
			VolumeClaimTemplateNames: volumeClaimTemplateNames,
		})
	}

	c.logger.Info("Successfully collected statefulsets", "count", len(statefulSets))
	return statefulSets, nil
}

// OpenShift-specific collectors
func collectRoutes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting routes", "namespace", namespace)
	// Note: OpenShift routes require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

func collectProjects(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting projects")
	// Note: OpenShift projects require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

func collectImages(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting images")
	// Note: OpenShift images require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

// Example new collector demonstrating how easy it is to add
func collectPods(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting pods", "namespace", namespace)

	podList, err := c.clients.Kubernetes.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	pods := make([]any, 0, len(podList.Items))
	for _, pod := range podList.Items {
		var containerImages []string
		for _, container := range pod.Spec.Containers {
			containerImages = append(containerImages, container.Image)
		}

		// Create a simple pod structure - you can extend this based on your needs
		podData := map[string]any{
			"name":             pod.Name,
			"namespace":        pod.Namespace,
			"labels":           pod.Labels,
			"annotations":      AnnotationsCleaner(pod.Annotations),
			"created_at":       pod.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			"phase":            string(pod.Status.Phase),
			"node_name":        pod.Spec.NodeName,
			"host_network":     pod.Spec.HostNetwork,
			"container_images": containerImages,
			"restart_policy":   string(pod.Spec.RestartPolicy),
		}

		pods = append(pods, podData)
	}

	c.logger.Info("Successfully collected pods", "count", len(pods))
	return pods, nil
}
