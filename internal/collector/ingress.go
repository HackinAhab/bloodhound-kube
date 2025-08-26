package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Ingress struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Class       string            `json:"class,omitempty"`
	Rules       []IngressRule     `json:"rules,omitempty"`
	TLS         []IngressTLS      `json:"tls,omitempty"`
	Status      IngressStatus     `json:"status,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type IngressRule struct {
	Host string                `json:"host,omitempty"`
	HTTP *HTTPIngressRuleValue `json:"http,omitempty"`
}

type HTTPIngressRuleValue struct {
	Paths []HTTPIngressPath `json:"paths"`
}

type HTTPIngressPath struct {
	Path     string         `json:"path,omitempty"`
	PathType string         `json:"path_type,omitempty"`
	Backend  IngressBackend `json:"backend"`
}

type IngressBackend struct {
	Service *IngressServiceBackend `json:"service,omitempty"`
}

type IngressServiceBackend struct {
	Name string             `json:"name"`
	Port IngressServicePort `json:"port"`
}

type IngressServicePort struct {
	Name   string `json:"name,omitempty"`
	Number int32  `json:"number,omitempty"`
}

type IngressTLS struct {
	Hosts      []string `json:"hosts,omitempty"`
	SecretName string   `json:"secret_name,omitempty"`
}

type IngressStatus struct {
	LoadBalancer LoadBalancerStatus `json:"load_balancer,omitempty"`
}

func (c *Collector) CollectIngresses(ctx context.Context, namespace string) ([]Ingress, error) {
	c.logger.Info("Collecting ingresses", "namespace", namespace)

	ingressList, err := c.client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	ingresses := make([]Ingress, 0, len(ingressList.Items))
	for _, ing := range ingressList.Items {
		var rules []IngressRule
		for _, rule := range ing.Spec.Rules {
			ingressRule := IngressRule{
				Host: rule.Host,
			}

			if rule.HTTP != nil {
				var paths []HTTPIngressPath
				for _, path := range rule.HTTP.Paths {
					ingressPath := HTTPIngressPath{
						Path: path.Path,
					}

					if path.PathType != nil {
						ingressPath.PathType = string(*path.PathType)
					}

					if path.Backend.Service != nil {
						ingressPath.Backend.Service = &IngressServiceBackend{
							Name: path.Backend.Service.Name,
							Port: IngressServicePort{
								Name:   path.Backend.Service.Port.Name,
								Number: path.Backend.Service.Port.Number,
							},
						}
					}

					paths = append(paths, ingressPath)
				}

				ingressRule.HTTP = &HTTPIngressRuleValue{Paths: paths}
			}

			rules = append(rules, ingressRule)
		}

		var tls []IngressTLS
		for _, t := range ing.Spec.TLS {
			tls = append(tls, IngressTLS{
				Hosts:      t.Hosts,
				SecretName: t.SecretName,
			})
		}

		var statusIngress []LoadBalancerIngress
		for _, lbIng := range ing.Status.LoadBalancer.Ingress {
			statusIngress = append(statusIngress, LoadBalancerIngress{
				IP:       lbIng.IP,
				Hostname: lbIng.Hostname,
			})
		}

		var ingressClass string
		if ing.Spec.IngressClassName != nil {
			ingressClass = *ing.Spec.IngressClassName
		}

		ingresses = append(ingresses, Ingress{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Class:     ingressClass,
			Rules:     rules,
			TLS:       tls,
			Status: IngressStatus{
				LoadBalancer: LoadBalancerStatus{
					Ingress: statusIngress,
				},
			},
			Labels:      ing.Labels,
			Annotations: ing.Annotations,
			CreatedAt:   ing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.logger.Info("Successfully collected ingresses", "count", len(ingresses))
	return ingresses, nil
}

type IngressesHandler struct {
	*BaseHandler
}

func NewIngressesHandler() *IngressesHandler {
	return &IngressesHandler{
		BaseHandler: &BaseHandler{
			name:          "ingresses",
			clusterScoped: false,
		},
	}
}

func (h *IngressesHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	ingresses, err := c.CollectIngresses(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(ingresses))

	for _, ingress := range ingresses {
		batch = append(batch, Resource{
			Type:      "ingress",
			Namespace: namespace,
			Resource:  ingress,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
