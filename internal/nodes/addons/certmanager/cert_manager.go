//go:build !no_addons && !no_cert_manager

package certmanager

import . "bloodhound-kube/internal/nodes/framework"

func Register() {
	RegisterKind("Certificate", BuildCertificateNode)
	RegisterKind("Issuer", BuildIssuerNode)
	RegisterKind("ClusterIssuer", BuildClusterIssuerNode)
}

func BuildCertificateNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	issuerRef := GetMap(spec, "issuerRef")
	secretName := GetString(spec, "secretName")
	issuerRefName := GetString(issuerRef, "name")
	issuerRefKind := GetStringDefault(issuerRef, "kind", "Issuer")

	properties := map[string]any{
		"name":          name,
		"namespace":     namespace,
		"labels":        MapToSortedList(labelsMap),
		"annotations":   MapToSortedList(annotationsMap),
		"secretName":    secretName,
		"issuerRefName": issuerRefName,
		"issuerRefKind": issuerRefKind,
		"dnsNames":      StringSliceFromAny(GetSlice(spec, "dnsNames")),
		"isCA":          spec["isCA"] == true,
	}

	base := NewGraphNodeBase("BHK_Certificate", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Certificate{
			GraphNodeBase: base,
			SecretName:    secretName,
			IssuerRefName: issuerRefName,
			IssuerRefKind: issuerRefKind,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func BuildIssuerNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	caSecretName, vaultSecretName := issuerSecretRefs(spec)

	properties := map[string]any{
		"name":            name,
		"namespace":       namespace,
		"labels":          MapToSortedList(labelsMap),
		"annotations":     MapToSortedList(annotationsMap),
		"issuerType":      issuerTypeFromSpec(spec),
		"caSecretName":    caSecretName,
		"vaultSecretName": vaultSecretName,
	}

	base := NewGraphNodeBase("BHK_Issuer", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Issuer{
			GraphNodeBase:   base,
			CASecretName:    caSecretName,
			VaultSecretName: vaultSecretName,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func BuildClusterIssuerNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	caSecretName, vaultSecretName := issuerSecretRefs(spec)

	properties := map[string]any{
		"name":            name,
		"namespace":       "",
		"labels":          MapToSortedList(labelsMap),
		"annotations":     MapToSortedList(annotationsMap),
		"issuerType":      issuerTypeFromSpec(spec),
		"caSecretName":    caSecretName,
		"vaultSecretName": vaultSecretName,
	}

	base := NewGraphNodeBase("BHK_ClusterIssuer", "", name, labelsMap, annotationsMap)

	core := CoreEntry{
		Cluster: true,
		Data: ClusterIssuer{
			GraphNodeBase:   base,
			CASecretName:    caSecretName,
			VaultSecretName: vaultSecretName,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

// issuerTypeFromSpec returns the first key under spec (ca, vault, acme,
// selfSigned, venafi) — mirrors providerTypeFromSpec in the externalsecrets
// addon.
func issuerTypeFromSpec(spec map[string]any) string {
	keys := MapKeysSorted(spec)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// issuerSecretRefs extracts the two highest attack-path-value secret
// references from an Issuer/ClusterIssuer spec: the CA issuer's signing key
// (spec.ca.secretName — compromise forges arbitrary certs trusted by this
// issuer) and the Vault issuer's auth credential (spec.vault.auth.*, checked
// across the three supported auth methods). ACME/SelfSigned issuers have no
// equivalent standing secret and are intentionally not parsed here.
func issuerSecretRefs(spec map[string]any) (caSecretName, vaultSecretName string) {
	caSecretName = GetString(GetMap(spec, "ca"), "secretName")

	auth := GetMap(GetMap(spec, "vault"), "auth")
	if name := GetString(GetMap(auth, "tokenSecretRef"), "name"); name != "" {
		return caSecretName, name
	}
	if name := GetString(GetMap(GetMap(auth, "appRole"), "secretRef"), "name"); name != "" {
		return caSecretName, name
	}
	if name := GetString(GetMap(GetMap(auth, "kubernetes"), "secretRef"), "name"); name != "" {
		return caSecretName, name
	}
	return caSecretName, ""
}
