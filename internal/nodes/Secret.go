package nodes

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"time"
)

type Secret struct {
	GraphNodeBase
	SecretType string
	Data       map[string]any
}

func init() {
	Register("Secret", BuildSecretNode)
}

func BuildSecretNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	data := GetMap(resource, "data")
	keys := MapKeysSorted(data)
	entries := MapEntriesSorted(data)

	secretType := GetString(resource, "type")
	// Helm release secrets have the "release" key in their data, and it is huge/unnecessary for this.
	if secretType == "helm.sh/release.v1" {
		entries[0] = "release={{Removed for clarity}}"
	}

	properties := map[string]any{
		"name":                  name,
		"namespace":             namespace,
		"labels":                MapToSortedList(labelsMap),
		"annotations":           MapToSortedList(annotationsMap),
		"secretType":            secretType,
		"dataKeys":              keys,
		"dataEntries":           entries,
		"isServiceAccountToken": secretType == "kubernetes.io/service-account-token",
		"isTlsSecret":           secretType == "kubernetes.io/tls",
		"isOpaque":              secretType == "Opaque" || secretType == "",
	}

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

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Secret{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Secret", namespace, name),
				Kinds:          []string{"Secret"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			SecretType: secretType,
			Data:       data,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Secret", namespace, name),
			Kinds:      []string{"Secret"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func parseTLSCertificate(data map[string]any) (map[string]any, bool) {
	certData, ok := data["tls.crt"]
	if !ok {
		return nil, false
	}
	certString, ok := certData.(string)
	if !ok || certString == "" {
		return nil, false
	}
	certBytes := []byte(certString)
	if decoded, err := base64.StdEncoding.DecodeString(certString); err == nil {
		certBytes = decoded
	}

	cert, err := parseCertificateBytes(certBytes)
	if err != nil {
		return nil, false
	}

	ipAddresses := make([]string, 0, len(cert.IPAddresses))
	for _, ip := range cert.IPAddresses {
		ipAddresses = append(ipAddresses, ip.String())
	}

	uris := make([]string, 0, len(cert.URIs))
	for _, uri := range cert.URIs {
		uris = append(uris, uri.String())
	}

	info := map[string]any{
		"subject":            cert.Subject.String(),
		"issuer":             cert.Issuer.String(),
		"commonName":         cert.Subject.CommonName,
		"notBefore":          cert.NotBefore.Format(time.RFC3339),
		"notAfter":           cert.NotAfter.Format(time.RFC3339),
		"dnsNames":           cert.DNSNames,
		"ipAddresses":        ipAddresses,
		"emailAddresses":     cert.EmailAddresses,
		"uris":               uris,
		"isCA":               cert.IsCA,
		"signatureAlgorithm": cert.SignatureAlgorithm.String(),
		"publicKeyAlgorithm": cert.PublicKeyAlgorithm.String(),
		"keyUsage":           keyUsageStrings(cert.KeyUsage),
		"extKeyUsage":        extKeyUsageStrings(cert.ExtKeyUsage),
	}

	return info, true
}

func parseCertificateBytes(certBytes []byte) (*x509.Certificate, error) {
	if block, _ := pem.Decode(certBytes); block != nil {
		return x509.ParseCertificate(block.Bytes)
	}
	return x509.ParseCertificate(certBytes)
}

func keyUsageStrings(usage x509.KeyUsage) []string {
	items := []string{}
	if usage&x509.KeyUsageDigitalSignature != 0 {
		items = append(items, "DigitalSignature")
	}
	if usage&x509.KeyUsageContentCommitment != 0 {
		items = append(items, "ContentCommitment")
	}
	if usage&x509.KeyUsageKeyEncipherment != 0 {
		items = append(items, "KeyEncipherment")
	}
	if usage&x509.KeyUsageDataEncipherment != 0 {
		items = append(items, "DataEncipherment")
	}
	if usage&x509.KeyUsageKeyAgreement != 0 {
		items = append(items, "KeyAgreement")
	}
	if usage&x509.KeyUsageCertSign != 0 {
		items = append(items, "CertSign")
	}
	if usage&x509.KeyUsageCRLSign != 0 {
		items = append(items, "CRLSign")
	}
	return items
}

func extKeyUsageStrings(usages []x509.ExtKeyUsage) []string {
	items := make([]string, 0, len(usages))
	for _, usage := range usages {
		switch usage {
		case x509.ExtKeyUsageServerAuth:
			items = append(items, "ServerAuth")
		case x509.ExtKeyUsageClientAuth:
			items = append(items, "ClientAuth")
		case x509.ExtKeyUsageCodeSigning:
			items = append(items, "CodeSigning")
		case x509.ExtKeyUsageEmailProtection:
			items = append(items, "EmailProtection")
		case x509.ExtKeyUsageTimeStamping:
			items = append(items, "TimeStamping")
		case x509.ExtKeyUsageOCSPSigning:
			items = append(items, "OCSPSigning")
		}
	}
	return items
}
