package edges

import "strings"

type parsedPerm struct {
	key          string
	verbs        map[string]struct{}
	resourceName string
}

// TODO: Build test cases for this helper.
func accessForResource(perms []string, resourceKeys []string, verbs []string) (bool, map[string]struct{}) {
	parsed := parseRBACPerms(perms)
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
