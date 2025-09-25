package nodes

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"maps"
	"strings"
	"time"

	"bloodhound-kube/internal/bloodhound"
	"bloodhound-kube/internal/collector"
)

type SecretPropertyMapper struct{}

func (m *SecretPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	resourceData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret resource: %w", err)
	}

	var rawSecret map[string]any
	if err := json.Unmarshal(resourceData, &rawSecret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw secret: %w", err)
	}

	var secret collector.Secret
	if err := json.Unmarshal(resourceData, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	properties := map[string]any{
		"secret_type":     secret.Type,
		"data_keys_count": len(secret.DataKeys),
		"created_at":      secret.CreatedAt,

		"is_service_account_token": secret.Type == "kubernetes.io/service-account-token",
		"is_tls_secret":            secret.Type == "kubernetes.io/tls",
		"is_registry_secret":       secret.Type == "kubernetes.io/dockercfg" || secret.Type == "kubernetes.io/dockerconfigjson",
		"is_opaque":                secret.Type == "Opaque",
	}

	if len(secret.DataKeys) > 0 {
		var sensitiveKeys []string
		hasPrivateKey := false
		hasPassword := false
		hasToken := false

		for _, key := range secret.DataKeys {
			lowerKey := strings.ToLower(key)

			if strings.Contains(lowerKey, "key") &&
				(strings.Contains(lowerKey, "private") || strings.Contains(lowerKey, "priv")) {
				hasPrivateKey = true
				sensitiveKeys = append(sensitiveKeys, key)
			} else if strings.Contains(lowerKey, "password") ||
				strings.Contains(lowerKey, "passwd") ||
				strings.Contains(lowerKey, "pass") {
				hasPassword = true
				sensitiveKeys = append(sensitiveKeys, key)
			} else if strings.Contains(lowerKey, "token") ||
				strings.Contains(lowerKey, "secret") ||
				strings.Contains(lowerKey, "key") {
				hasToken = true
				sensitiveKeys = append(sensitiveKeys, key)
			}
		}

		properties["has_private_key"] = hasPrivateKey
		properties["has_password"] = hasPassword
		properties["has_token"] = hasToken

		properties["sensitive_keys_count"] = len(sensitiveKeys)
		properties["has_sensitive_keys"] = len(sensitiveKeys) > 0
	}

	if secret.Labels != nil {
		if sa, exists := secret.Labels["kubernetes.io/service-account.name"]; exists {
			properties["service_account"] = sa
		}

		if secret.Annotations != nil {
			if managedBy, exists := secret.Annotations["kubectl.kubernetes.io/last-applied-configuration"]; exists && managedBy != "" {
				properties["managed_by_kubectl"] = true
			}
		}
	}

	// Process secret data based on type
	secretDataInfo := m.processSecretData(secret)
	maps.Copy(properties, secretDataInfo)

	// TLS Certificate Analysis for Internal Discovery
	if secret.Type == "kubernetes.io/tls" {
		properties["has_tls_cert"] = false
		properties["has_tls_key"] = false

		for _, key := range secret.DataKeys {
			if key == "tls.crt" {
				properties["has_tls_cert"] = true
			}
			if key == "tls.key" {
				properties["has_tls_key"] = true
			}
		}

		tlsInfo := m.extractTLSInformation(rawSecret)
		for key, value := range tlsInfo {
			properties[key] = value
		}

		domainInfo := m.inferDomainsFromTLSSecret(secret)
		for key, value := range domainInfo {
			properties[key] = value
		}
	}

	return properties, nil
}

// processSecretData handles secret data based on type - includes non-TLS data decoded, TLS data omitted for clarity
func (m *SecretPropertyMapper) processSecretData(secret collector.Secret) map[string]any {
	dataInfo := make(map[string]any)

	if secret.Type == "kubernetes.io/tls" {
		// For TLS secrets, always show that data is omitted for clarity
		dataInfo["tls_data_status"] = "omitted for clarity"

		// Try to parse certificate for domain names if data is available
		if len(secret.Data) > 0 {
			if certData, exists := secret.Data["tls.crt"]; exists {
				if domains := m.extractDomainsFromTLSCert([]byte(certData)); len(domains) > 0 {
					dataInfo["tls_cert_domains"] = domains
				}
			}
		}

		// If data is not available but keys exist, at least note what keys are present
		if len(secret.Data) == 0 && len(secret.DataKeys) > 0 {
			dataInfo["tls_data_keys"] = secret.DataKeys
		}
	} else {
		// For non-TLS secrets, add each secret key as its own property with uppercase name
		if len(secret.Data) > 0 {
			for key, value := range secret.Data {
				// Try to base64 decode the string value in case it's actually base64-encoded data
				decodedBytes, err := base64.StdEncoding.DecodeString(value)
				var decodedValue string
				if err != nil {
					// If decoding fails, the value is likely already plain text
					decodedValue = value
				} else {
					// If decoding succeeds, use the decoded value
					decodedValue = string(decodedBytes)
				}

				// Check if the data looks like it might be sensitive (contains certain patterns)
				lowerKey := strings.ToLower(key)
				if m.isSensitiveKey(lowerKey) {
					// For sensitive keys, we might want to truncate or mask the data
					if len(decodedValue) > 50 {
						decodedValue = decodedValue[:50] + "...[truncated]"
					}
				}

				// Use uppercase key name as property key
				propertyKey := strings.ToUpper(key)
				dataInfo[propertyKey] = decodedValue
			}
		} else {
			// If no actual data but we have keys, add placeholder to show structure
			for _, key := range secret.DataKeys {
				propertyKey := strings.ToUpper(key)
				dataInfo[propertyKey] = "[data not available during collection]"
			}
		}
	}

	return dataInfo
}

// extractDomainsFromTLSCert extracts domain names from a TLS certificate
func (m *SecretPropertyMapper) extractDomainsFromTLSCert(certData []byte) []string {
	var domains []string

	block, _ := pem.Decode(certData)
	if block == nil {
		return domains
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return domains
	}

	if cert.Subject.CommonName != "" {
		domains = append(domains, cert.Subject.CommonName)
	}

	domains = append(domains, cert.DNSNames...)

	return domains
}

// isSensitiveKey determines if a key likely contains sensitive data
func (m *SecretPropertyMapper) isSensitiveKey(key string) bool {
	sensitivePatterns := []string{
		"password", "passwd", "pass", "secret", "token", "key", "auth",
		"credential", "cert", "private", "api", "access", "refresh",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}

	return false
}

// extractTLSInformation parses TLS certificate data for domain discovery
func (m *SecretPropertyMapper) extractTLSInformation(rawSecret map[string]any) map[string]any {
	tlsProperties := make(map[string]any)

	data, ok := rawSecret["data"].(map[string]any)
	if !ok {
		return tlsProperties
	}

	var certData, keyData string
	if cert, exists := data["tls.crt"]; exists {
		if certStr, ok := cert.(string); ok {
			certData = certStr
		}
	}
	if key, exists := data["tls.key"]; exists {
		if keyStr, ok := key.(string); ok {
			keyData = keyStr
		}
	}

	tlsProperties["has_tls_cert"] = certData != ""
	tlsProperties["has_tls_key"] = keyData != ""

	if certData != "" {
		certInfo := m.parseTLSCertificate(certData)
		for key, value := range certInfo {
			tlsProperties[key] = value
		}
	}

	return tlsProperties
}

// parseTLSCertificate extracts domain and security information from TLS certificates
func (m *SecretPropertyMapper) parseTLSCertificate(certData string) map[string]any {
	certInfo := make(map[string]any)

	certBytes, err := base64.StdEncoding.DecodeString(certData)
	if err != nil {
		certInfo["cert_decode_error"] = true
		return certInfo
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		certInfo["cert_parse_error"] = true
		return certInfo
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		certInfo["cert_x509_error"] = true
		return certInfo
	}

	var domains []string
	var internalDomains []string
	var externalDomains []string

	if cert.Subject.CommonName != "" {
		domains = append(domains, cert.Subject.CommonName)
		if m.isInternalDomain(cert.Subject.CommonName) {
			internalDomains = append(internalDomains, cert.Subject.CommonName)
		} else {
			externalDomains = append(externalDomains, cert.Subject.CommonName)
		}
	}

	for _, name := range cert.DNSNames {
		domains = append(domains, name)
		if m.isInternalDomain(name) {
			internalDomains = append(internalDomains, name)
		} else {
			externalDomains = append(externalDomains, name)
		}
	}

	var ipAddresses []string
	for _, ip := range cert.IPAddresses {
		ipAddresses = append(ipAddresses, ip.String())
	}

	certInfo["cert_domains_count"] = len(domains)
	certInfo["cert_internal_domains_count"] = len(internalDomains)
	certInfo["cert_external_domains_count"] = len(externalDomains)
	certInfo["cert_ip_addresses_count"] = len(ipAddresses)
	certInfo["cert_common_name"] = cert.Subject.CommonName
	certInfo["cert_issuer"] = cert.Issuer.CommonName
	certInfo["cert_not_before"] = cert.NotBefore.Format("2006-01-02T15:04:05Z")
	certInfo["cert_not_after"] = cert.NotAfter.Format("2006-01-02T15:04:05Z")
	now := time.Now()
	certInfo["cert_is_expired"] = now.After(cert.NotAfter)
	certInfo["cert_is_valid_now"] = now.After(cert.NotBefore) && now.Before(cert.NotAfter)
	certInfo["cert_is_self_signed"] = cert.Issuer.CommonName == cert.Subject.CommonName
	certInfo["cert_signature_algorithm"] = cert.SignatureAlgorithm.String()
	certInfo["cert_key_usage"] = cert.KeyUsage

	hasWildcard := false
	for _, domain := range domains {
		if strings.HasPrefix(domain, "*.") {
			hasWildcard = true
			break
		}
	}
	certInfo["cert_has_wildcard"] = hasWildcard

	var securityIssues []string
	if now.After(cert.NotAfter) {
		securityIssues = append(securityIssues, "expired_certificate")
	}
	if cert.Issuer.CommonName == cert.Subject.CommonName {
		securityIssues = append(securityIssues, "self_signed_certificate")
	}
	certInfo["cert_security_issues_count"] = len(securityIssues)
	certInfo["has_security_issues"] = len(securityIssues) > 0
	if len(securityIssues) > 0 {
		certInfo["has_expired_cert"] = strings.Contains(strings.Join(securityIssues, ","), "expired_certificate")
		certInfo["has_self_signed_cert"] = strings.Contains(strings.Join(securityIssues, ","), "self_signed_certificate")
	}

	return certInfo
}

// isInternalDomain determines if a domain is likely internal for discovery purposes
func (m *SecretPropertyMapper) isInternalDomain(domain string) bool {
	internalPatterns := []string{
		".local", ".internal", ".corp", ".company", ".lan",
		".cluster", ".k8s", ".kube", ".svc", ".default",
		"localhost", "127.0.0.1", "::1",
	}

	lowerDomain := strings.ToLower(domain)

	for _, pattern := range internalPatterns {
		if strings.Contains(lowerDomain, pattern) {
			return true
		}
	}

	if strings.Contains(lowerDomain, "10.") ||
		strings.Contains(lowerDomain, "192.168.") ||
		strings.Contains(lowerDomain, "172.") {
		return true
	}

	return false
}

// inferDomainsFromTLSSecret extracts domain information from TLS secret metadata
func (m *SecretPropertyMapper) inferDomainsFromTLSSecret(secret collector.Secret) map[string]any {
	domainInfo := make(map[string]any)
	var inferredDomains []string
	var internalDomains []string
	var externalDomains []string

	name := strings.ToLower(secret.Name)

	namePatterns := []string{
		"-tls", "-cert", "-ssl", "-wildcard",
	}

	cleanName := name
	for _, pattern := range namePatterns {
		cleanName = strings.ReplaceAll(cleanName, pattern, "")
	}

	if strings.Contains(cleanName, "-") {
		potentialDomain := strings.ReplaceAll(cleanName, "-", ".")
		if m.isValidDomainPattern(potentialDomain) {
			inferredDomains = append(inferredDomains, potentialDomain)
			if m.isInternalDomain(potentialDomain) {
				internalDomains = append(internalDomains, potentialDomain)
			} else {
				externalDomains = append(externalDomains, potentialDomain)
			}
		}
	}

	if strings.Contains(name, "wildcard") {
		parts := strings.Split(name, "-")
		for i, part := range parts {
			if part == "wildcard" && i > 0 {
				baseName := parts[i-1]
				if len(baseName) > 2 {
					// For wildcard certs, often the name pattern is "domain-wildcard-tls"
					potentialDomain := "*." + baseName + ".local" // Common internal pattern
					inferredDomains = append(inferredDomains, potentialDomain)
					internalDomains = append(internalDomains, potentialDomain)

					regularDomain := baseName + ".local"
					inferredDomains = append(inferredDomains, regularDomain)
					internalDomains = append(internalDomains, regularDomain)
				}
				break
			}
		}
	}

	if secret.Annotations != nil {
		for key, value := range secret.Annotations {
			lowerKey := strings.ToLower(key)
			lowerValue := strings.ToLower(value)

			if strings.Contains(lowerKey, "domain") ||
				strings.Contains(lowerKey, "host") ||
				strings.Contains(lowerKey, "dns") {
				domains := m.extractDomainsFromString(value)
				inferredDomains = append(inferredDomains, domains...)
				for _, domain := range domains {
					if m.isInternalDomain(domain) {
						internalDomains = append(internalDomains, domain)
					} else {
						externalDomains = append(externalDomains, domain)
					}
				}
			}

			if strings.Contains(lowerKey, "cert-manager") && strings.Contains(lowerValue, ".") {
				domains := m.extractDomainsFromString(value)
				inferredDomains = append(inferredDomains, domains...)
				for _, domain := range domains {
					if m.isInternalDomain(domain) {
						internalDomains = append(internalDomains, domain)
					} else {
						externalDomains = append(externalDomains, domain)
					}
				}
			}
		}
	}

	if len(inferredDomains) > 0 {
		domainInfo["inferred_domains_count"] = len(inferredDomains)
		domainInfo["inferred_internal_domains_count"] = len(internalDomains)
		domainInfo["inferred_external_domains_count"] = len(externalDomains)
		domainInfo["has_inferred_domains"] = true
	} else {
		domainInfo["has_inferred_domains"] = false
	}

	hasWildcard := false
	for _, domain := range inferredDomains {
		if strings.Contains(domain, "wildcard") || strings.HasPrefix(domain, "*.") {
			hasWildcard = true
			break
		}
	}
	domainInfo["inferred_has_wildcard"] = hasWildcard

	return domainInfo
}

// extractDomainsFromString extracts potential domains from a string value
func (m *SecretPropertyMapper) extractDomainsFromString(value string) []string {
	var domains []string

	parts := strings.FieldsFunc(value, func(c rune) bool {
		return c == ' ' || c == ',' || c == ';' || c == '|'
	})

	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if m.isValidDomainPattern(cleaned) {
			domains = append(domains, cleaned)
		}
	}

	return domains
}

// isValidDomainPattern checks if a string looks like a domain name
func (m *SecretPropertyMapper) isValidDomainPattern(s string) bool {
	if len(s) < 3 {
		return false
	}

	if !strings.Contains(s, ".") {
		return false
	}

	validChars := "abcdefghijklmnopqrstuvwxyz0123456789-.*"
	for _, char := range strings.ToLower(s) {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}

type SecretParser struct {
	config bloodhound.ResourceConfig
}

func NewSecretParser() *SecretParser {
	return &SecretParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "secret",
			PrimaryKind:    "Secret",
			PropertyMapper: &SecretPropertyMapper{},
		},
	}
}

func (p *SecretParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *SecretParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *SecretParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *SecretParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	secretData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret resource: %w", err)
	}

	var secret collector.Secret
	if err := json.Unmarshal(secretData, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		secret.Name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
