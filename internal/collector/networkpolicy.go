package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkPolicy struct {
	Name        string                     `json:"name"`
	Namespace   string                     `json:"namespace"`
	Labels      map[string]string          `json:"labels,omitempty"`
	Annotations map[string]string          `json:"annotations,omitempty"`
	CreatedAt   string                     `json:"created_at"`
	PodSelector NetworkPolicyLabelSelector `json:"pod_selector"`
	PolicyTypes []string                   `json:"policy_types,omitempty"`
	Ingress     []NetworkPolicyIngressRule `json:"ingress,omitempty"`
	Egress      []NetworkPolicyEgressRule  `json:"egress,omitempty"`
}

type NetworkPolicyLabelSelector struct {
	MatchLabels      map[string]string               `json:"match_labels,omitempty"`
	MatchExpressions []NetworkPolicyLabelSelectorReq `json:"match_expressions,omitempty"`
}

type NetworkPolicyLabelSelectorReq struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type NetworkPolicyIngressRule struct {
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
	From  []NetworkPolicyPeer `json:"from,omitempty"`
}

type NetworkPolicyEgressRule struct {
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
	To    []NetworkPolicyPeer `json:"to,omitempty"`
}

type NetworkPolicyPort struct {
	Protocol string `json:"protocol,omitempty"`
	Port     string `json:"port,omitempty"`
	EndPort  *int32 `json:"end_port,omitempty"`
}

type NetworkPolicyPeer struct {
	PodSelector       *NetworkPolicyLabelSelector `json:"pod_selector,omitempty"`
	NamespaceSelector *NetworkPolicyLabelSelector `json:"namespace_selector,omitempty"`
	IPBlock           *NetworkPolicyIPBlock       `json:"ip_block,omitempty"`
}

type NetworkPolicyIPBlock struct {
	CIDR   string   `json:"cidr"`
	Except []string `json:"except,omitempty"`
}

func (c *Collector) CollectNetworkPolicies(ctx context.Context, namespace string) ([]NetworkPolicy, error) {
	c.logger.Info("Collecting networkpolicies", "namespace", namespace)

	networkPolicyList, err := c.clients.Kubernetes.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networkpolicies: %w", err)
	}

	networkPolicies := make([]NetworkPolicy, 0, len(networkPolicyList.Items))
	for _, np := range networkPolicyList.Items {
		podSelector := NetworkPolicyLabelSelector{
			MatchLabels: np.Spec.PodSelector.MatchLabels,
		}
		for _, expr := range np.Spec.PodSelector.MatchExpressions {
			podSelector.MatchExpressions = append(podSelector.MatchExpressions, NetworkPolicyLabelSelectorReq{
				Key:      expr.Key,
				Operator: string(expr.Operator),
				Values:   expr.Values,
			})
		}

		var policyTypes []string
		for _, pt := range np.Spec.PolicyTypes {
			policyTypes = append(policyTypes, string(pt))
		}

		var ingressRules []NetworkPolicyIngressRule
		for _, rule := range np.Spec.Ingress {
			ingressRule := NetworkPolicyIngressRule{}

			for _, port := range rule.Ports {
				npPort := NetworkPolicyPort{}
				if port.Protocol != nil {
					npPort.Protocol = string(*port.Protocol)
				}
				if port.Port != nil {
					npPort.Port = port.Port.String()
				}
				if port.EndPort != nil {
					npPort.EndPort = port.EndPort
				}
				ingressRule.Ports = append(ingressRule.Ports, npPort)
			}
			for _, peer := range rule.From {
				npPeer := NetworkPolicyPeer{}

				if peer.PodSelector != nil {
					podSel := &NetworkPolicyLabelSelector{
						MatchLabels: peer.PodSelector.MatchLabels,
					}
					for _, expr := range peer.PodSelector.MatchExpressions {
						podSel.MatchExpressions = append(podSel.MatchExpressions, NetworkPolicyLabelSelectorReq{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					npPeer.PodSelector = podSel
				}

				if peer.NamespaceSelector != nil {
					nsSel := &NetworkPolicyLabelSelector{
						MatchLabels: peer.NamespaceSelector.MatchLabels,
					}
					for _, expr := range peer.NamespaceSelector.MatchExpressions {
						nsSel.MatchExpressions = append(nsSel.MatchExpressions, NetworkPolicyLabelSelectorReq{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					npPeer.NamespaceSelector = nsSel
				}

				if peer.IPBlock != nil {
					npPeer.IPBlock = &NetworkPolicyIPBlock{
						CIDR:   peer.IPBlock.CIDR,
						Except: peer.IPBlock.Except,
					}
				}

				ingressRule.From = append(ingressRule.From, npPeer)
			}

			ingressRules = append(ingressRules, ingressRule)
		}

		var egressRules []NetworkPolicyEgressRule
		for _, rule := range np.Spec.Egress {
			egressRule := NetworkPolicyEgressRule{}

			for _, port := range rule.Ports {
				npPort := NetworkPolicyPort{}
				if port.Protocol != nil {
					npPort.Protocol = string(*port.Protocol)
				}
				if port.Port != nil {
					npPort.Port = port.Port.String()
				}
				if port.EndPort != nil {
					npPort.EndPort = port.EndPort
				}
				egressRule.Ports = append(egressRule.Ports, npPort)
			}

			for _, peer := range rule.To {
				npPeer := NetworkPolicyPeer{}

				if peer.PodSelector != nil {
					podSel := &NetworkPolicyLabelSelector{
						MatchLabels: peer.PodSelector.MatchLabels,
					}
					for _, expr := range peer.PodSelector.MatchExpressions {
						podSel.MatchExpressions = append(podSel.MatchExpressions, NetworkPolicyLabelSelectorReq{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					npPeer.PodSelector = podSel
				}

				if peer.NamespaceSelector != nil {
					nsSel := &NetworkPolicyLabelSelector{
						MatchLabels: peer.NamespaceSelector.MatchLabels,
					}
					for _, expr := range peer.NamespaceSelector.MatchExpressions {
						nsSel.MatchExpressions = append(nsSel.MatchExpressions, NetworkPolicyLabelSelectorReq{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					npPeer.NamespaceSelector = nsSel
				}

				if peer.IPBlock != nil {
					npPeer.IPBlock = &NetworkPolicyIPBlock{
						CIDR:   peer.IPBlock.CIDR,
						Except: peer.IPBlock.Except,
					}
				}

				egressRule.To = append(egressRule.To, npPeer)
			}

			egressRules = append(egressRules, egressRule)
		}

		networkPolicies = append(networkPolicies, NetworkPolicy{
			Name:        np.Name,
			Namespace:   np.Namespace,
			Labels:      np.Labels,
			Annotations: AnnotationsCleaner(np.Annotations),
			CreatedAt:   np.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			PodSelector: podSelector,
			PolicyTypes: policyTypes,
			Ingress:     ingressRules,
			Egress:      egressRules,
		})
	}

	c.logger.Info("Successfully collected networkpolicies", "count", len(networkPolicies))
	return networkPolicies, nil
}

type NetworkPoliciesHandler struct {
	*BaseHandler
}

func NewNetworkPoliciesHandler() *NetworkPoliciesHandler {
	return &NetworkPoliciesHandler{
		BaseHandler: &BaseHandler{
			name:          "networkpolicies",
			clusterScoped: false,
		},
	}
}

func (h *NetworkPoliciesHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	networkPolicies, err := c.CollectNetworkPolicies(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(networkPolicies))

	for _, networkPolicy := range networkPolicies {
		batch = append(batch, Resource{
			Type:      "networkpolicy",
			Namespace: namespace,
			Resource:  networkPolicy,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
