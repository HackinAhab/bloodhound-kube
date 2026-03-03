package edges

import "bloodhound-kube/internal/model"

type sccEdgesRule struct{}

func (r sccEdgesRule) Name() string {
	return "security_context_constraints"
}

func init() {
	RegisterEdgeRule(sccEdgesRule{})
}

func (r sccEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Pods {
			pod := &space.Pods[i]
			if pod.AnnotationsMap == nil {
				continue
			}
			sccName, _ := pod.AnnotationsMap["openshift.io/scc"].(string)
			if sccName == "" {
				continue
			}
			if scc := ctx.Index.SecurityContextConstraintsBy[sccName]; scc != nil {
				edges = append(edges, CreateEdge(scc, pod, "EnforcedSCC"))
			}
		}
	}
	return edges
}
