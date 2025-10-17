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
	c.logger.Debug("Starting configmap collection", "namespace", namespace)

	configMapList, err := c.clients.Kubernetes.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list configmaps", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	c.logger.Debug("Retrieved configmap list", "namespace", namespace, "count", len(configMapList.Items))

	configMaps := make([]any, 0, len(configMapList.Items))
	for _, cm := range configMapList.Items {
		var dataKeys []string
		var dataMap map[string]string
		var binaryDataKeys []string
		var binaryDataMap map[string][]byte

		if c.IsRedacted() {
			// When redacted, collect key names but redact values
			for key := range cm.Data {
				dataKeys = append(dataKeys, key)
			}
			for key := range cm.BinaryData {
				binaryDataKeys = append(binaryDataKeys, key)
			}
			dataMap = nil
			binaryDataMap = nil
		} else {
			// Normal collection - include keys and data
			dataMap = make(map[string]string)
			for key, value := range cm.Data {
				dataKeys = append(dataKeys, key)
				dataMap[key] = value
			}

			binaryDataMap = make(map[string][]byte)
			for key, value := range cm.BinaryData {
				binaryDataKeys = append(binaryDataKeys, key)
				binaryDataMap[key] = value
			}
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

	c.logger.Info("Successfully collected configmaps", "namespace", namespace, "count", len(configMaps))
	c.logger.Debug("Configmap collection completed", "namespace", namespace, "processed", len(configMaps))
	return configMaps, nil
}

// collectSecrets collects Kubernetes Secrets from the specified namespace
func collectSecrets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting secrets", "namespace", namespace)
	c.logger.Debug("Starting secret collection", "namespace", namespace)

	secretList, err := c.clients.Kubernetes.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list secrets", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	c.logger.Debug("Retrieved secret list", "namespace", namespace, "count", len(secretList.Items))

	secrets := make([]any, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var dataKeys []string
		var dataMap map[string]string

		if c.IsRedacted() {
			// When redacted, collect key names but redact values
			for key := range secret.Data {
				dataKeys = append(dataKeys, key)
			}
			dataMap = nil
		} else {
			dataMap = make(map[string]string)
			for key, value := range secret.Data {
				dataKeys = append(dataKeys, key)
				dataMap[key] = string(value)
			}
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

	c.logger.Info("Successfully collected secrets", "namespace", namespace, "count", len(secrets))
	c.logger.Debug("Secret collection completed", "namespace", namespace, "processed", len(secrets))
	return secrets, nil
}

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

// collectServices collects Kubernetes Services from the specified namespace
func collectServices(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting services", "namespace", namespace)
	c.logger.Debug("Starting service collection", "namespace", namespace)

	serviceList, err := c.clients.Kubernetes.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list services", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	c.logger.Debug("Retrieved service list", "namespace", namespace, "count", len(serviceList.Items))

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

	c.logger.Info("Successfully collected services", "namespace", namespace, "count", len(services))
	c.logger.Debug("Service collection completed", "namespace", namespace, "processed", len(services))
	return services, nil
}

// collectIngresses collects Kubernetes Ingresses from the specified namespace
func collectIngresses(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting ingresses", "namespace", namespace)
	c.logger.Debug("Starting ingress collection", "namespace", namespace)

	ingressList, err := c.clients.Kubernetes.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list ingresses", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	c.logger.Debug("Retrieved ingress list", "namespace", namespace, "count", len(ingressList.Items))

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

	c.logger.Info("Successfully collected ingresses", "namespace", namespace, "count", len(ingresses))
	c.logger.Debug("Ingress collection completed", "namespace", namespace, "processed", len(ingresses))
	return ingresses, nil
}

// collectGateways collects Gateway API resources from the specified namespace
func collectGateways(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting gateways", "namespace", namespace)
	c.logger.Debug("Gateway collection not yet implemented", "namespace", namespace)
	//TODO
	return []any{}, nil
}

// collectRBAC collects RBAC resources (cluster-scoped: Roles, ClusterRoles, RoleBindings, ClusterRoleBindings)
func collectRBAC(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting RBAC resources")
	c.logger.Debug("Starting RBAC collection", "namespace", namespace)

	var rbacResources []any

	if namespace != "" {
		// Collect namespaced RBAC resources for specific namespace
		rbacResources = append(rbacResources, collectRoles(ctx, c, namespace)...)
		rbacResources = append(rbacResources, collectRoleBindings(ctx, c, namespace)...)
	} else {
		// Collect cluster-scoped RBAC resources
		rbacResources = append(rbacResources, collectClusterRoles(ctx, c)...)
		rbacResources = append(rbacResources, collectClusterRoleBindings(ctx, c)...)

		// Also collect all namespaced RBAC resources when doing cluster-wide collection
		rbacResources = append(rbacResources, collectRoles(ctx, c, "")...)
		rbacResources = append(rbacResources, collectRoleBindings(ctx, c, "")...)
	}

	c.logger.Info("Successfully collected RBAC resources", "count", len(rbacResources))
	c.logger.Debug("RBAC collection completed", "processed", len(rbacResources))
	return rbacResources, nil
}

// collectRoles collects Roles from the specified namespace (or all namespaces if namespace is empty)
func collectRoles(ctx context.Context, c *Collector, namespace string) []any {
	roleList, err := c.clients.Kubernetes.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var roles []any
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

		roles = append(roles, RBACResource{
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
	return roles
}

// collectRoleBindings collects RoleBindings from the specified namespace (or all namespaces if namespace is empty)
func collectRoleBindings(ctx context.Context, c *Collector, namespace string) []any {
	roleBindingList, err := c.clients.Kubernetes.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var roleBindings []any
	for _, rb := range roleBindingList.Items {
		var subjects []RBACSubject
		for _, subject := range rb.Subjects {
			subjects = append(subjects, RBACSubject{
				Kind:      subject.Kind,
				Name:      subject.Name,
				Namespace: subject.Namespace,
			})
		}

		roleBindings = append(roleBindings, RBACResource{
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
	return roleBindings
}

// collectClusterRoles collects ClusterRoles (cluster-scoped)
func collectClusterRoles(ctx context.Context, c *Collector) []any {
	clusterRoleList, err := c.clients.Kubernetes.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var clusterRoles []any
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

		clusterRoles = append(clusterRoles, RBACResource{
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
	return clusterRoles
}

// collectClusterRoleBindings collects ClusterRoleBindings (cluster-scoped)
func collectClusterRoleBindings(ctx context.Context, c *Collector) []any {
	clusterRoleBindingList, err := c.clients.Kubernetes.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []any{}
	}

	var clusterRoleBindings []any
	for _, crb := range clusterRoleBindingList.Items {
		var subjects []RBACSubject
		for _, subject := range crb.Subjects {
			subjects = append(subjects, RBACSubject{
				Kind:      subject.Kind,
				Name:      subject.Name,
				Namespace: subject.Namespace,
			})
		}

		clusterRoleBindings = append(clusterRoleBindings, RBACResource{
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
	return clusterRoleBindings
}

// collectNetworkPolicies collects Kubernetes NetworkPolicies from the specified namespace
func collectNetworkPolicies(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting network policies", "namespace", namespace)
	c.logger.Debug("Starting network policy collection", "namespace", namespace)

	npList, err := c.clients.Kubernetes.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list network policies", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list network policies: %w", err)
	}

	c.logger.Debug("Retrieved network policy list", "namespace", namespace, "count", len(npList.Items))

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

	c.logger.Info("Successfully collected network policies", "namespace", namespace, "count", len(networkPolicies))
	c.logger.Debug("Network policy collection completed", "namespace", namespace, "processed", len(networkPolicies))
	return networkPolicies, nil
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

// OpenShift-specific collectors
func collectRoutes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting routes", "namespace", namespace)
	c.logger.Debug("Route collection not yet implemented (requires OpenShift client)", "namespace", namespace)
	// Note: OpenShift routes require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

func collectProjects(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting projects")
	c.logger.Debug("Project collection not yet implemented (requires OpenShift client)")
	// Note: OpenShift projects require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

func collectImages(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting images")
	c.logger.Debug("Image collection not yet implemented (requires OpenShift client)")
	// Note: OpenShift images require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

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
