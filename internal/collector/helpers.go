package collector

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"slices"
	"strings"
	"time"
)

// This file contains helper functions for certificate parsing and sensitive data detection.
// These functions provide value-added analysis beyond simple Kubernetes API collection.

// CertificateInfo represents parsed certificate metadata

const stripManagedFieldsEnabled = true
const omitHelmReleaseSecrets = true

var commonSensitiveKeys = []string{
	"tls.key", "ca.key", "key.pem", "private.key", "server.key", "client.key",
	"tls-key", "ca-key", "private-key", "server-key", "client-key",
	"password", "token", "secret", "api-key", "apikey", "auth",
}

var commonCertificateKeys = []string{
	"tls.crt", "ca.crt", "cert.pem", "certificate.pem", "client.crt", "server.crt",
	"tls-cert", "ca-cert", "certificate", "cert", "ca", "client-cert", "server-cert",
}

type CertificateInfo struct {
	Subject        string    `json:"subject,omitempty"`
	Issuer         string    `json:"issuer,omitempty"`
	NotBefore      time.Time `json:"not_before,omitempty"`
	NotAfter       time.Time `json:"not_after,omitempty"`
	DNSNames       []string  `json:"dns_names,omitempty"`
	IPAddresses    []string  `json:"ip_addresses,omitempty"`
	EmailAddresses []string  `json:"email_addresses,omitempty"`
	URIs           []string  `json:"uris,omitempty"`
	IsCA           bool      `json:"is_ca,omitempty"`
	KeyUsage       []string  `json:"key_usage,omitempty"`
	ExtKeyUsage    []string  `json:"ext_key_usage,omitempty"`
}

// parseCertificate parses a PEM-encoded certificate and extracts metadata
func parseCertificate(certPEM string) (*CertificateInfo, error) {
	// Decode PEM block
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Convert IP addresses to strings
	var ipAddresses []string
	for _, ip := range cert.IPAddresses {
		ipAddresses = append(ipAddresses, ip.String())
	}

	// Convert URIs to strings
	var uris []string
	for _, uri := range cert.URIs {
		uris = append(uris, uri.String())
	}

	// Convert key usage flags to strings
	var keyUsage []string
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		keyUsage = append(keyUsage, "DigitalSignature")
	}
	if cert.KeyUsage&x509.KeyUsageContentCommitment != 0 {
		keyUsage = append(keyUsage, "ContentCommitment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		keyUsage = append(keyUsage, "KeyEncipherment")
	}
	if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
		keyUsage = append(keyUsage, "DataEncipherment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
		keyUsage = append(keyUsage, "KeyAgreement")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		keyUsage = append(keyUsage, "CertSign")
	}
	if cert.KeyUsage&x509.KeyUsageCRLSign != 0 {
		keyUsage = append(keyUsage, "CRLSign")
	}

	// Convert extended key usage to strings
	var extKeyUsage []string
	for _, eku := range cert.ExtKeyUsage {
		switch eku {
		case x509.ExtKeyUsageServerAuth:
			extKeyUsage = append(extKeyUsage, "ServerAuth")
		case x509.ExtKeyUsageClientAuth:
			extKeyUsage = append(extKeyUsage, "ClientAuth")
		case x509.ExtKeyUsageCodeSigning:
			extKeyUsage = append(extKeyUsage, "CodeSigning")
		case x509.ExtKeyUsageEmailProtection:
			extKeyUsage = append(extKeyUsage, "EmailProtection")
		case x509.ExtKeyUsageTimeStamping:
			extKeyUsage = append(extKeyUsage, "TimeStamping")
		case x509.ExtKeyUsageOCSPSigning:
			extKeyUsage = append(extKeyUsage, "OCSPSigning")
		}
	}

	return &CertificateInfo{
		Subject:        cert.Subject.String(),
		Issuer:         cert.Issuer.String(),
		NotBefore:      cert.NotBefore,
		NotAfter:       cert.NotAfter,
		DNSNames:       cert.DNSNames,
		IPAddresses:    ipAddresses,
		EmailAddresses: cert.EmailAddresses,
		URIs:           uris,
		IsCA:           cert.IsCA,
		KeyUsage:       keyUsage,
		ExtKeyUsage:    extKeyUsage,
	}, nil
}

// extractCertificatesFromSecret extracts and parses certificates from secret data
func extractCertificatesFromSecret(data map[string][]byte) map[string]CertificateInfo {
	certificates := make(map[string]CertificateInfo)

	for _, key := range commonCertificateKeys {
		if certData, exists := data[key]; exists {
			if certInfo, err := parseCertificate(string(certData)); err == nil {
				certificates[key] = *certInfo
			}
		}
	}

	// Also check for any key ending with .crt or .pem
	for key, certData := range data {
		if strings.HasSuffix(key, ".crt") || strings.HasSuffix(key, ".pem") {
			if _, exists := certificates[key]; !exists {
				if certInfo, err := parseCertificate(string(certData)); err == nil {
					certificates[key] = *certInfo
				}
			}
		}
	}

	return certificates
}

// isSensitiveKey returns true if the key often contains sensitive data that should be redacted
func isSensitiveKey(key string) bool {

	keyLower := strings.ToLower(key)
	for _, sensitive := range commonSensitiveKeys {
		if strings.Contains(keyLower, sensitive) {
			return true
		}
	}
	return false
}

func annotationsCleaner(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	cleaned := make(map[string]string)
	for key, value := range annotations {
		// Skip the kubectl last-applied-configuration and revision annotations, it's extremely large and not often useful
		// TODO: Make this configurable.
		if key != "kubectl.kubernetes.io/last-applied-configuration" && key != "deployment.kubernetes.io/revision" {
			cleaned[key] = value
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

// cleanAnnotationsInObject applies annotation cleaning on a Kubernetes object map
func cleanAnnotationsInObject(obj map[string]any) {
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok || metadata == nil {
		return
	}

	annotationsRaw, ok := metadata["annotations"].(map[string]any)
	if !ok || annotationsRaw == nil {
		return
	}

	annotations := make(map[string]string, len(annotationsRaw))
	for key, value := range annotationsRaw {
		if str, ok := value.(string); ok {
			annotations[key] = str
		}
	}

	cleaned := annotationsCleaner(annotations)
	if cleaned == nil {
		delete(metadata, "annotations")
		return
	}

	cleanedAny := make(map[string]any, len(cleaned))
	for key, value := range cleaned {
		cleanedAny[key] = value
	}
	metadata["annotations"] = cleanedAny
}

func getSecretDataBytes(obj map[string]any) map[string][]byte {
	dataRaw, ok := obj["data"].(map[string]any)
	if !ok || dataRaw == nil {
		return nil
	}

	dataBytes := make(map[string][]byte, len(dataRaw))
	for key, value := range dataRaw {
		str, ok := value.(string)
		if !ok {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			continue
		}
		dataBytes[key] = decoded
	}

	if len(dataBytes) == 0 {
		return nil
	}
	return dataBytes
}

// enrichSecretObject adds certificate metadata and optional redaction to a Secret object map
func enrichSecretObject(obj map[string]any, redacted bool) {
	dataBytes := getSecretDataBytes(obj)
	if dataBytes != nil {
		certs := extractCertificatesFromSecret(dataBytes)
		if len(certs) > 0 {
			obj["certificates"] = certs
		}
	}

	if !redacted {
		return
	}

	dataRaw, ok := obj["data"].(map[string]any)
	if !ok || dataRaw == nil {
		return
	}

	redactedKeys := make([]string, 0)
	for key := range dataRaw {
		if isSensitiveKey(key) {
			delete(dataRaw, key)
			redactedKeys = append(redactedKeys, key)
		}
	}

	if len(redactedKeys) > 0 {
		obj["redacted_keys"] = redactedKeys
	}
	if len(dataRaw) == 0 {
		delete(obj, "data")
	}
}

// applyCollectionHelpers applies common cleaning/enrichment to a collected object
func applyCollectionHelpers(obj map[string]any, resourcePlural string, redacted bool) map[string]any {
	if obj == nil {
		return obj
	}
	if resourcePlural == "secrets" && omitHelmReleaseSecrets && isHelmReleaseSecret(obj) {
		return nil
	}

	stripManagedFields(obj)

	cleanAnnotationsInObject(obj)

	if resourcePlural == "secrets" {
		enrichSecretObject(obj, redacted)
	}

	return obj
}

func stripManagedFields(obj map[string]any) {
	if !stripManagedFieldsEnabled {
		return
	}
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok || metadata == nil {
		return
	}
	delete(metadata, "managedFields")
}

func isHelmReleaseSecret(obj map[string]any) bool {
	secretType, ok := obj["type"].(string)
	if !ok {
		return false
	}
	return secretType == "helm.sh/release.v1"
}

func normalizeResourceType(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func buildAPIPath(groupVersion, resource string) string {
	if groupVersion == "" {
		return resource
	}
	return groupVersion + "/" + resource
}

func hasVerb(verbs []string, verb string) bool {
	return slices.Contains(verbs, verb)
}
