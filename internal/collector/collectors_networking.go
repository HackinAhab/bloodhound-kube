package collector

import (
	"context"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

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

// collectHTTPRoutes collects Gateway API HTTPRoutes from the specified namespace
func collectHTTPRoutes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting HTTPRoutes", "namespace", namespace)
	c.logger.Debug("Starting HTTPRoute collection", "namespace", namespace)

	// Gateway API support is not yet implemented in the client
	c.logger.Debug("Gateway API client not yet implemented, skipping HTTPRoute collection", "namespace", namespace)
	c.logger.Info("HTTPRoute collection skipped - Gateway API support not available", "namespace", namespace)
	// TODO
	return []any{}, nil
}

// collectGRPCRoutes collects Gateway API GRPCRoutes from the specified namespace
func collectGRPCRoutes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting GRPCRoutes", "namespace", namespace)
	c.logger.Debug("Starting GRPCRoute collection", "namespace", namespace)

	// Gateway API support is not yet implemented in the client
	c.logger.Debug("Gateway API client not yet implemented, skipping GRPCRoute collection", "namespace", namespace)
	c.logger.Info("GRPCRoute collection skipped - Gateway API support not available", "namespace", namespace)
	// TODO
	return []any{}, nil
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
