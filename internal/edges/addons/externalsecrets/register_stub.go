//go:build no_addons || no_external_secrets

package externalsecrets

import "bloodhound-kube/internal/edges/framework"

// Register is a no-op when external-secrets support isn't compiled in — see
// external_secrets.go for the real implementation.
func Register(reg *framework.Registry) {}
