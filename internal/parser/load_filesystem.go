//go:build !embedded

package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"bloodhound-kube/internal/utils"
	"github.com/open-policy-agent/opa/v1/ast"
)

func loadPolicyModules(policyDir string, recursive bool) (*ast.Compiler, []string, error) {
	if _, err := os.Stat(policyDir); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("policy directory does not exist: %s", policyDir)
	}

	var policyFiles []string
	if recursive {
		err := filepath.Walk(policyDir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(filePath) == ".rego" {
				policyFiles = append(policyFiles, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find policy files: %w", err)
		}
	} else {
		matches, err := filepath.Glob(filepath.Join(policyDir, "*.rego"))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find policy files: %w", err)
		}
		policyFiles = matches
	}

	if len(policyFiles) == 0 {
		return nil, nil, fmt.Errorf("no policy files found in %s", policyDir)
	}

	utils.DefaultLogger().Debug("Loaded policy files", "count", len(policyFiles), "directory", policyDir, "recursive", recursive)

	return nil, policyFiles, nil
}
