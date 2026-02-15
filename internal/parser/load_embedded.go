//go:build embedded

package parser

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"bloodhound-kube/internal/utils"
	regopolicies "bloodhound-kube/rego"

	"github.com/open-policy-agent/opa/v1/ast"
)

func loadPolicyModules(policyDir string, recursive bool) (*ast.Compiler, []string, error) {
	cleanDir := strings.TrimPrefix(policyDir, "./")
	cleanDir = path.Clean(cleanDir)
	cleanDir = strings.TrimPrefix(cleanDir, "rego/")
	if cleanDir == "rego" {
		cleanDir = "."
	}

	if _, err := fs.Stat(regopolicies.FS, cleanDir); err != nil {
		return nil, nil, fmt.Errorf("policy directory does not exist: %s", policyDir)
	}

	modules := make(map[string]string)
	err := fs.WalkDir(regopolicies.FS, cleanDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path.Ext(p) != ".rego" {
			return nil
		}
		if !recursive && path.Dir(p) != cleanDir {
			return nil
		}
		data, err := fs.ReadFile(regopolicies.FS, p)
		if err != nil {
			return err
		}
		modules[p] = string(data)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find policy files: %w", err)
	}

	if len(modules) == 0 {
		return nil, nil, fmt.Errorf("no policy files found in %s", policyDir)
	}
	utils.DefaultLogger().Debug("Loaded policy modules", "count", len(modules), "directory", policyDir, "recursive", recursive, "embedded", true)

	compiler, err := ast.CompileModules(modules)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile policies: %w", err)
	}

	if compiler.Failed() {
		return nil, nil, compiler.Errors
	}

	return compiler, nil, nil
}
