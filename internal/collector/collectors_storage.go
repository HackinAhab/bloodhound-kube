package collector

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file contains collection functions for Kubernetes storage resources:
// ConfigMaps, Secrets, and related collection logic.

// collectConfigMaps collects Kubernetes ConfigMaps from the specified namespace
func collectConfigMaps(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting configmaps", "namespace", namespace)
	c.logger.Debug("Starting configmap collection", "namespace", namespace)

	configMapList, err := c.clients.Kubernetes.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list configmaps", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	c.logger.Debug("Retrieved configmap list", "namespace", namespace, "count", len(configMapList.Items))

	configMaps := make([]any, 0, len(configMapList.Items))
	for _, cm := range configMapList.Items {
		var dataKeys []string
		var dataMap map[string]string
		var binaryDataKeys []string
		var binaryDataMap map[string][]byte

		if c.IsRedacted() {
			// When redacted, collect key names but redact values
			for key := range cm.Data {
				dataKeys = append(dataKeys, key)
			}
			for key := range cm.BinaryData {
				binaryDataKeys = append(binaryDataKeys, key)
			}
			dataMap = nil
			binaryDataMap = nil
		} else {
			// Normal collection - include keys and data
			dataMap = make(map[string]string)
			for key, value := range cm.Data {
				dataKeys = append(dataKeys, key)
				dataMap[key] = value
			}

			binaryDataMap = make(map[string][]byte)
			for key, value := range cm.BinaryData {
				binaryDataKeys = append(binaryDataKeys, key)
				binaryDataMap[key] = value
			}
		}

		configMaps = append(configMaps, ConfigMap{
			CommonResourceMeta: CommonResourceMeta{
				Name:        cm.Name,
				Namespace:   cm.Namespace,
				Labels:      cm.Labels,
				Annotations: AnnotationsCleaner(cm.Annotations),
				CreatedAt:   cm.CreationTimestamp.Time,
			},
			DataKeys:       dataKeys,
			Data:           dataMap,
			BinaryDataKeys: binaryDataKeys,
			BinaryData:     binaryDataMap,
		})
	}

	c.logger.Info("Successfully collected configmaps", "namespace", namespace, "count", len(configMaps))
	c.logger.Debug("Configmap collection completed", "namespace", namespace, "processed", len(configMaps))
	return configMaps, nil
}

// collectSecrets collects Kubernetes Secrets from the specified namespace
func collectSecrets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting secrets", "namespace", namespace)
	c.logger.Debug("Starting secret collection", "namespace", namespace)

	secretList, err := c.clients.Kubernetes.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list secrets", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	c.logger.Debug("Retrieved secret list", "namespace", namespace, "count", len(secretList.Items))

	secrets := make([]any, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var dataKeys []string
		var dataMap map[string]string

		certificates := extractCertificatesFromSecret(secret.Data)

		if c.IsRedacted() {
			// When redacted, collect key names but redact sensitive values
			for key := range secret.Data {
				dataKeys = append(dataKeys, key)
			}

			// Create redacted data map
			dataMap = make(map[string]string)
			for key, value := range secret.Data {
				if isSensitiveKey(key) {
					dataMap[key] = "[REDACTED]"
				} else {
					// For likely non-sensitive data, include the data
					dataMap[key] = string(value)
				}
			}
		} else {
			dataMap = make(map[string]string)
			for key, value := range secret.Data {
				dataKeys = append(dataKeys, key)
				dataMap[key] = string(value)
			}
		}

		secretResource := Secret{
			CommonResourceMeta: CommonResourceMeta{
				Name:        secret.Name,
				Namespace:   secret.Namespace,
				Labels:      secret.Labels,
				Annotations: AnnotationsCleaner(secret.Annotations),
				CreatedAt:   secret.CreationTimestamp.Time,
			},
			Type:     string(secret.Type),
			DataKeys: dataKeys,
			Data:     dataMap,
		}

		// Only include certificates map if we found any
		if len(certificates) > 0 {
			secretResource.Certificates = certificates
		}

		secrets = append(secrets, secretResource)
	}

	c.logger.Info("Successfully collected secrets", "namespace", namespace, "count", len(secrets))
	c.logger.Debug("Secret collection completed", "namespace", namespace, "processed", len(secrets))
	return secrets, nil
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

	// Common certificate keys to check
	certKeys := []string{
		"tls.crt", "ca.crt", "cert.pem", "certificate.pem", "client.crt", "server.crt",
		"tls-cert", "ca-cert", "certificate", "cert", "ca", "client-cert", "server-cert",
	}

	for _, key := range certKeys {
		if certData, exists := data[key]; exists {
			if certInfo, err := parseCertificate(string(certData)); err == nil {
				certificates[key] = *certInfo
			}
		}
	}

	// Also check for any key ending with .crt or .pem that we haven't processed yet
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
	sensitiveKeys := []string{
		"tls.key", "ca.key", "key.pem", "private.key", "server.key", "client.key",
		"tls-key", "ca-key", "private-key", "server-key", "client-key",
		"password", "token", "secret", "api-key", "apikey", "auth",
	}

	keyLower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(keyLower, sensitive) {
			return true
		}
	}
	return false
}
