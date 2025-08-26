package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Service struct {
	Name         string             `json:"name"`
	Namespace    string             `json:"namespace"`
	Type         string             `json:"type"`
	ClusterIP    string             `json:"cluster_ip,omitempty"`
	ExternalIPs  []string           `json:"external_ips,omitempty"`
	LoadBalancer LoadBalancerStatus `json:"load_balancer,omitempty"`
	Ports        []ServicePort      `json:"ports,omitempty"`
	Selector     map[string]string  `json:"selector,omitempty"`
	Labels       map[string]string  `json:"labels,omitempty"`
	Annotations  map[string]string  `json:"annotations,omitempty"`
	CreatedAt    string             `json:"created_at"`
}

type ServicePort struct {
	Name       string             `json:"name,omitempty"`
	Protocol   string             `json:"protocol"`
	Port       int32              `json:"port"`
	TargetPort intstr.IntOrString `json:"target_port"`
	NodePort   int32              `json:"node_port,omitempty"`
}

type LoadBalancerStatus struct {
	Ingress []LoadBalancerIngress `json:"ingress,omitempty"`
}

type LoadBalancerIngress struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

func (c *Collector) CollectServices(ctx context.Context, namespace string) ([]Service, error) {
	c.logger.Info("Collecting services", "namespace", namespace)

	serviceList, err := c.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services := make([]Service, 0, len(serviceList.Items))
	for _, svc := range serviceList.Items {
		var ports []ServicePort
		for _, port := range svc.Spec.Ports {
			ports = append(ports, ServicePort{
				Name:       port.Name,
				Protocol:   string(port.Protocol),
				Port:       port.Port,
				TargetPort: port.TargetPort,
				NodePort:   port.NodePort,
			})
		}

		var ingress []LoadBalancerIngress
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			ingress = append(ingress, LoadBalancerIngress{
				IP:       ing.IP,
				Hostname: ing.Hostname,
			})
		}

		services = append(services, Service{
			Name:        svc.Name,
			Namespace:   svc.Namespace,
			Type:        string(svc.Spec.Type),
			ClusterIP:   svc.Spec.ClusterIP,
			ExternalIPs: svc.Spec.ExternalIPs,
			LoadBalancer: LoadBalancerStatus{
				Ingress: ingress,
			},
			Ports:       ports,
			Selector:    svc.Spec.Selector,
			Labels:      svc.Labels,
			Annotations: svc.Annotations,
			CreatedAt:   svc.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.logger.Info("Successfully collected services", "count", len(services))
	return services, nil
}

type ServicesHandler struct {
	*BaseHandler
}

func NewServicesHandler() *ServicesHandler {
	return &ServicesHandler{
		BaseHandler: &BaseHandler{
			name:          "services",
			clusterScoped: false,
		},
	}
}

func (h *ServicesHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	services, err := c.CollectServices(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(services))

	for _, service := range services {
		batch = append(batch, Resource{
			Type:      "service",
			Namespace: namespace,
			Resource:  service,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
