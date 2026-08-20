package parser

import (
	"encoding/json"
	"sort"
	"strings"

	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
)

// enrichIdentityPermissions resolves all RoleBindings and ClusterRoleBindings,
// aggregates the unique set of RbacRules granted to each identity subject, and
// writes the result as a "permissions" property on the corresponding node.
func enrichIdentityPermissions(nodeList []model.BloodHoundNode, coreFacts *model.CoreFacts) {
	if coreFacts == nil {
		return
	}

	index := model.NewEdgeIndex(coreFacts)

	// nodeByID maps a node's pre-cluster ID to its position in nodeList so we
	// can patch its Properties in place.
	nodeByID := make(map[string]int, len(nodeList))
	for i, n := range nodeList {
		nodeByID[n.ID] = i
	}

	// accumulated maps identity node ID → set of unique RbacRules.
	// The key for dedup is the tuple (APIGroup, Resource, sorted verbs, sorted resourceNames).
	type ruleKey struct {
		apiGroup      string
		resource      string
		verbs         string
		resourceNames string
	}
	accumulated := map[string]map[ruleKey]nodefw.RbacRule{}

	addRules := func(nodeID string, rules []nodefw.RbacRule) {
		if nodeID == "" || len(rules) == 0 {
			return
		}
		if _, ok := accumulated[nodeID]; !ok {
			accumulated[nodeID] = map[ruleKey]nodefw.RbacRule{}
		}
		set := accumulated[nodeID]
		for _, rule := range rules {
			verbsCopy := append([]string(nil), rule.Verbs...)
			sort.Strings(verbsCopy)
			namesCopy := append([]string(nil), rule.ResourceNames...)
			sort.Strings(namesCopy)

			k := ruleKey{
				apiGroup:      rule.APIGroup,
				resource:      rule.Resource,
				verbs:         strings.Join(verbsCopy, ","),
				resourceNames: strings.Join(namesCopy, ","),
			}
			if _, exists := set[k]; !exists {
				set[k] = nodefw.RbacRule{
					APIGroup:      rule.APIGroup,
					Resource:      rule.Resource,
					Verbs:         verbsCopy,
					ResourceNames: namesCopy,
				}
			}
		}
	}

	rulesForRole := func(namespace, roleKind, roleName string) []nodefw.RbacRule {
		switch roleKind {
		case "Role":
			ns := index.RolesByNamespace[namespace]
			if ns == nil {
				return nil
			}
			role := ns[roleName]
			if role == nil {
				return nil
			}
			return role.Rules
		case "ClusterRole":
			cr := index.ClusterRolesByName[roleName]
			if cr == nil {
				return nil
			}
			return cr.Rules
		}
		return nil
	}

	subjectNodeID := func(namespace string, subject nodefw.Subject) string {
		switch subject.Kind {
		case "ServiceAccount":
			ns := subject.Namespace
			if ns == "" {
				ns = namespace
			}
			if ns == "" {
				return ""
			}
			return nodefw.BuildID("BHK_ServiceAccount", ns, subject.Name)
		case "User":
			return nodefw.BuildID("BHK_User", "", subject.Name)
		case "Group":
			return nodefw.BuildID("BHK_Group", "", subject.Name)
		}
		return ""
	}

	for ns, space := range coreFacts.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.RoleBindings {
			b := &space.RoleBindings[i]
			rules := rulesForRole(ns, b.RoleKind, b.RoleName)
			if len(rules) == 0 {
				continue
			}
			for _, subject := range b.Subjects {
				id := subjectNodeID(ns, subject)
				addRules(id, rules)
			}
		}
	}

	for i := range coreFacts.Cluster.ClusterRoleBindings {
		b := &coreFacts.Cluster.ClusterRoleBindings[i]
		rules := rulesForRole("", b.RoleKind, b.RoleName)
		if len(rules) == 0 {
			continue
		}
		for _, subject := range b.Subjects {
			id := subjectNodeID("", subject)
			addRules(id, rules)
		}
	}

	for nodeID, ruleSet := range accumulated {
		idx, ok := nodeByID[nodeID]
		if !ok {
			continue
		}
		rules := make([]nodefw.RbacRule, 0, len(ruleSet))
		for _, r := range ruleSet {
			rules = append(rules, r)
		}
		sort.Slice(rules, func(i, j int) bool {
			a, b := rules[i], rules[j]
			if a.APIGroup != b.APIGroup {
				return a.APIGroup < b.APIGroup
			}
			if a.Resource != b.Resource {
				return a.Resource < b.Resource
			}
			return strings.Join(a.Verbs, ",") < strings.Join(b.Verbs, ",")
		})

		perms := make([]string, 0, len(rules))
		for _, r := range rules {
			entry := map[string]any{
				"apiGroup": r.APIGroup,
				"resource": r.Resource,
				"verbs":    r.Verbs,
			}
			if len(r.ResourceNames) > 0 {
				entry["resourceNames"] = r.ResourceNames
			}
			b, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			perms = append(perms, string(b))
		}

		if nodeList[idx].Properties == nil {
			nodeList[idx].Properties = map[string]any{}
		}
		nodeList[idx].Properties["permissions"] = perms
	}
}
