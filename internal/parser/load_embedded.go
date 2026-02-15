//go:build embedded

package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"bloodhound-kube/internal/utils"
	regopolicies "bloodhound-kube/rego"

	"github.com/open-policy-agent/opa/v1/ast"
)

func loadPolicyModules(policyDir string, recursive bool, extraDirs []string) (*ast.Compiler, []string, error) {
	log := utils.DefaultLogger().Component("parser")
	cleanDir := strings.TrimPrefix(policyDir, "./")
	cleanDir = path.Clean(cleanDir)
	cleanDir = strings.TrimPrefix(cleanDir, "rego/")
	if cleanDir == "rego" {
		cleanDir = "."
	}
	scope := policyScopeFromDir(cleanDir)

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
	log.Debug("Loaded policy modules", "count", len(modules), "directory", policyDir, "recursive", recursive, "embedded", true)

	if err := mergeExtraPolicyModules(modules, extraDirs, log, scope); err != nil {
		return nil, nil, err
	}

	compiler, err := ast.CompileModules(modules)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile policies: %w", err)
	}

	if compiler.Failed() {
		return nil, nil, compiler.Errors
	}

	return compiler, nil, nil
}

func mergeExtraPolicyModules(modules map[string]string, extraDirs []string, log utils.Logger, scope string) error {
	for _, dir := range extraDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil {
			log.Error("Extra policy directory not accessible", "directory", dir, "error", err)
			return fmt.Errorf("extra policy directory not accessible: %s", dir)
		}
		if !info.IsDir() {
			log.Error("Extra policy path is not a directory", "directory", dir)
			return fmt.Errorf("extra policy path is not a directory: %s", dir)
		}

		err = filepath.WalkDir(dir, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(filePath) != ".rego" {
				return nil
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			logicalPath := normalizePolicyPath(dir, filePath)
			if !shouldIncludePolicy(scope, logicalPath) {
				return nil
			}
			if _, exists := modules[logicalPath]; exists {
				log.Warn("Policy overridden", "path", logicalPath, "override_path", filePath)
			}
			modules[logicalPath] = string(data)
			return nil
		})
		if err != nil {
			log.Error("Failed to load extra policy directory", "directory", dir, "error", err)
			return fmt.Errorf("failed to load extra policy directory: %s", dir)
		}
	}

	return nil
}
