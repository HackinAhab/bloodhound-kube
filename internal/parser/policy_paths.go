package parser

import (
	"path/filepath"
	"strings"
)

func normalizePolicyPath(baseDir, filePath string) string {
	cleanBase := filepath.Clean(baseDir)
	cleanFile := filepath.Clean(filePath)

	fileSlash := filepath.ToSlash(cleanFile)
	if idx := strings.LastIndex(fileSlash, "/rego/"); idx != -1 {
		rel := fileSlash[idx+len("/rego/"):]
		return normalizePolicyPrefix(cleanBase, rel)
	}

	rel, err := filepath.Rel(cleanBase, cleanFile)
	if err != nil {
		rel = filepath.Base(cleanFile)
	}
	rel = filepath.ToSlash(rel)
	return normalizePolicyPrefix(cleanBase, rel)
}

func normalizePolicyPrefix(baseDir, relativePath string) string {
	relativePath = strings.TrimPrefix(relativePath, "./")
	if strings.HasPrefix(relativePath, "edges/") || strings.HasPrefix(relativePath, "nodes/") {
		return relativePath
	}

	baseName := filepath.Base(baseDir)
	if baseName == "edges" || baseName == "nodes" {
		return baseName + "/" + relativePath
	}

	return relativePath
}

func policyScopeFromDir(policyDir string) string {
	clean := filepath.ToSlash(filepath.Clean(policyDir))
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimPrefix(clean, "rego/")
	if strings.HasPrefix(clean, "nodes") {
		return "nodes"
	}
	if strings.HasPrefix(clean, "edges") {
		return "edges"
	}
	return ""
}

func shouldIncludePolicy(scope, logicalPath string) bool {
	if scope == "" {
		return true
	}
	if strings.HasPrefix(logicalPath, "nodes/") {
		return scope == "nodes"
	}
	if strings.HasPrefix(logicalPath, "edges/") {
		return scope == "edges"
	}
	return true
}
