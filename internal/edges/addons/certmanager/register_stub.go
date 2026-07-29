//go:build no_addons || no_cert_manager

package certmanager

import "bloodhound-kube/internal/edges/framework"

// Register is a no-op when cert-manager support isn't compiled in — see
// cert_manager.go for the real implementation.
func Register(reg *framework.Registry) {}
