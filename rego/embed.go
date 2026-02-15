//go:build embedded

package regopolicies

import "embed"

//go:embed edges nodes
var FS embed.FS
