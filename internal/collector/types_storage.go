package collector

import (
	"time"
)

// This file contains type definitions for Kubernetes storage resources:
// ConfigMaps, Secrets, and related structures.

// CertificateInfo represents parsed certificate metadata
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

type ConfigMap struct {
	CommonResourceMeta
	DataKeys       []string          `json:"data_keys"`
	Data           map[string]string `json:"data,omitempty"`
	BinaryDataKeys []string          `json:"binary_data_keys,omitempty"`
	BinaryData     map[string][]byte `json:"binary_data,omitempty"`
}

type Secret struct {
	CommonResourceMeta
	Type         string                     `json:"type"`
	DataKeys     []string                   `json:"data_keys"`
	Data         map[string]string          `json:"data,omitempty"` // Made optional since data might be redacted
	Certificates map[string]CertificateInfo `json:"certificates,omitempty"`
}
