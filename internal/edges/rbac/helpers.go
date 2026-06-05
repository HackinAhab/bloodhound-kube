package rbac

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/nodes/rbac"
)

func permsForBinding(ctx *framework.Context, namespace, roleKind, roleName string) []string {
	if ctx == nil {
		return nil
	}
	switch roleKind {
	case "Role":
		roleIndex := ctx.Index.RolesByNamespace[namespace]
		if roleIndex == nil {
			return nil
		}
		role := roleIndex[roleName]
		if role == nil {
			return nil
		}
		return role.PermsDisplay
	case "ClusterRole":
		clusterRole := ctx.Index.ClusterRolesByName[roleName]
		if clusterRole == nil {
			return nil
		}
		return clusterRole.PermsDisplay
	default:
		return nil
	}
}

func resolveNamespacedSubjectSA(ctx *framework.Context, namespace, bindingNamespace, subjectKind, subjectNamespace, subjectName string) *rbac.ServiceAccount {
	if ctx == nil || subjectKind != "ServiceAccount" {
		return nil
	}
	subjectNS := subjectNamespace
	if subjectNS == "" {
		subjectNS = bindingNamespace
	}
	if subjectNS != namespace {
		return nil
	}
	saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNS]
	if saIndex == nil {
		return nil
	}
	return saIndex[subjectName]
}

func resolveClusterSubjectSA(ctx *framework.Context, subjectKind, subjectNamespace, subjectName string) *rbac.ServiceAccount {
	if ctx == nil || subjectKind != "ServiceAccount" || subjectNamespace == "" {
		return nil
	}
	saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNamespace]
	if saIndex == nil {
		return nil
	}
	return saIndex[subjectName]
}

func hasName(names map[string]struct{}, name string) bool {
	if len(names) == 0 {
		return false
	}
	_, ok := names[name]
	return ok
}

type parsedPerm struct {
	key   string
	verbs map[string]struct{}
}

func accessForResource(perms []string, resourceKeys []string, verbs []string) (bool, map[string]struct{}) {
	return accessForParsedResource(parseRBACPerms(perms), resourceKeys, verbs)
}

func accessForParsedResource(parsed []parsedPerm, resourceKeys []string, verbs []string) (bool, map[string]struct{}) {
	var names map[string]struct{}
	for _, perm := range parsed {
		if !verbsCheck(perm.verbs, verbs) {
			continue
		}
		if perm.key == "*" {
			return true, nil
		}
		for _, key := range resourceKeys {
			if perm.key == key {
				return true, nil
			}
			if after, ok := strings.CutPrefix(perm.key, key+"/"); ok {
				if after == "" || strings.Contains(after, "/") {
					continue
				}
				if names == nil {
					names = map[string]struct{}{}
				}
				names[after] = struct{}{}
			}
		}
	}
	return false, names
}

func parseRBACPerms(perms []string) []parsedPerm {
	if len(perms) == 0 {
		return nil
	}
	parsed := make([]parsedPerm, 0, len(perms))
	for _, entry := range perms {
		parts := strings.SplitN(entry, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		verbs := strings.Split(parts[1], ", ")
		verbSet := map[string]struct{}{}
		for _, verb := range verbs {
			verb = strings.TrimSpace(verb)
			if verb == "" {
				continue
			}
			verbSet[verb] = struct{}{}
		}
		if key == "" || len(verbSet) == 0 {
			continue
		}
		parsed = append(parsed, parsedPerm{key: key, verbs: verbSet})
	}
	return parsed
}

func verbsCheck(verbSet map[string]struct{}, verbs []string) bool {
	if verbSet == nil {
		return false
	}
	if _, ok := verbSet["*"]; ok {
		return true
	}
	for _, verb := range verbs {
		if _, ok := verbSet[verb]; ok {
			return true
		}
	}
	return false
}
