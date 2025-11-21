package collector

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

// This file contains helper functions for certificate parsing and sensitive data detection.
// These functions provide value-added analysis beyond simple Kubernetes API collection.

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
