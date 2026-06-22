package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Secret struct {
	GraphNodeBase
	SecretType         string
	Data               map[string]any
	ServiceAccountName string
}

func BuildSecretNode(obj runtime.Object) (BuildResult, bool) {
	secret, ok := obj.(*corev1.Secret)
	if !ok || secret == nil {
		return BuildResult{}, false
	}
	name := secret.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := secret.Namespace
	labelsMap := StringMapToAnyMap(secret.Labels)
	annotationsMap := StringMapToAnyMap(secret.Annotations)
	saName := ""
	if value, ok := annotationsMap["kubernetes.io/service-account.name"].(string); ok {
		saName = value
	}

	secretType := string(secret.Type)
	decodeOpaqueValues := secretType == "Opaque" || secretType == ""

	data := secretDataToAnyMap(secret, decodeOpaqueValues)
	keys := MapKeysSorted(data)
	entries := MapToSortedList(data)

	if secretType == "helm.sh/release.v1" {
		for i, entry := range entries {
			if len(entry) >= 8 && entry[:8] == "release=" {
				entries[i] = "release={{Removed for clarity}}"
				break
			}
		}
	}

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["secretType"] = secretType
	properties["dataKeys"] = keys
	properties["dataEntries"] = entries
	properties["isServiceAccountToken"] = secretType == "kubernetes.io/service-account-token"
	properties["isTlsSecret"] = secretType == "kubernetes.io/tls"
	properties["isOpaque"] = secretType == "Opaque" || secretType == ""

	if secretType == "kubernetes.io/tls" {
		if tlsInfo, ok := parseTLSCertificate(data); ok {
			properties["isCA"] = tlsInfo["isCA"]
			properties["commonName"] = tlsInfo["commonName"]
			properties["dnsNames"] = tlsInfo["dnsNames"]
			properties["extKeyUsage"] = tlsInfo["extKeyUsage"]
			properties["issuer"] = tlsInfo["issuer"]
			properties["keyUsage"] = tlsInfo["keyUsage"]
			properties["notAfter"] = tlsInfo["notAfter"]
			properties["notBefore"] = tlsInfo["notBefore"]
			properties["publicKeyAlgorithm"] = tlsInfo["publicKeyAlgorithm"]
			properties["signatureAlgorithm"] = tlsInfo["signatureAlgorithm"]
			properties["subject"] = tlsInfo["subject"]
			properties["uris"] = tlsInfo["uris"]
		}
	}

	base := NewGraphNodeBase("BHK_Secret", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Secret{
			GraphNodeBase:      base,
			SecretType:         secretType,
			Data:               data,
			ServiceAccountName: saName,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
