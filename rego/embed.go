//go:build embedded

package regopolicies

import "embed"

//go:embed **/*.rego
var FS embed.FS
