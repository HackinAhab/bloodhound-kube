//go:build !embedded

package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bloodhound-kube/internal/utils"
	"github.com/open-policy-agent/opa/v1/ast"
)

func loadPolicyModules(policyDir string, recursive bool, extraDirs []string) (*ast.Compiler, []string, error) {
	log := utils.DefaultLogger().Component("parser")
	scope := policyScopeFromDir(policyDir)
	if _, err := os.Stat(policyDir); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("policy directory does not exist: %s", policyDir)
	}

	policyFiles := make(map[string]string)
	addPolicyFile := func(baseDir, filePath string) {
		logicalPath := normalizePolicyPath(baseDir, filePath)
		if !shouldIncludePolicy(scope, logicalPath) {
			return
		}
		if _, exists := policyFiles[logicalPath]; exists {
			log.Warn("Policy overridden", "path", logicalPath, "override_path", filePath)
		}
		policyFiles[logicalPath] = filePath
	}

	if recursive {
		err := filepath.Walk(policyDir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(filePath) == ".rego" {
				addPolicyFile(policyDir, filePath)
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
		for _, filePath := range matches {
			addPolicyFile(policyDir, filePath)
		}
	}

	for _, dir := range extraDirs {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "." || dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil {
			log.Error("Extra policy directory not accessible", "directory", dir, "error", err)
			return nil, nil, fmt.Errorf("extra policy directory not accessible: %s", dir)
		}
		if !info.IsDir() {
			log.Error("Extra policy path is not a directory", "directory", dir)
			return nil, nil, fmt.Errorf("extra policy path is not a directory: %s", dir)
		}
		err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(filePath) == ".rego" {
				addPolicyFile(dir, filePath)
			}
			return nil
		})
		if err != nil {
			log.Error("Failed to load extra policy directory", "directory", dir, "error", err)
			return nil, nil, fmt.Errorf("failed to load extra policy directory: %s", dir)
		}
	}

	if len(policyFiles) == 0 {
		return nil, nil, fmt.Errorf("no policy files found in %s", policyDir)
	}

	utils.DefaultLogger().Debug("Loaded policy files", "count", len(policyFiles), "directory", policyDir, "recursive", recursive)

	files := make([]string, 0, len(policyFiles))
	for _, filePath := range policyFiles {
		files = append(files, filePath)
	}
	sort.Strings(files)

	return nil, files, nil
}
