package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// goPackageInfo represents the JSON output from `go list -json`
type goPackageInfo struct {
	Dir        string `json:"Dir"`
	ImportPath string `json:"ImportPath"`
	Standard   bool   `json:"Standard"`
	Module     *struct {
		Path string `json:"Path"`
		Dir  string `json:"Dir"`
	} `json:"Module"`
	GoFiles []string `json:"GoFiles"`
}

// GoResolver resolves Go import paths to source files using `go list`
type GoResolver struct {
	// ModulePath is the current Go module path (from go.mod)
	// Used to determine if imports are local vs external
	ModulePath string
}

// NewGoResolver creates a new Go import resolver
// sourceFile should be a .go file within the module to determine the module root
func NewGoResolver(sourceFile string) (*GoResolver, error) {
	// Find module root and module path
	modPath, err := findGoModulePath(sourceFile)
	if err != nil {
		// Not in a module, that's OK for stdlib imports
		modPath = ""
	}

	return &GoResolver{
		ModulePath: modPath,
	}, nil
}

// GetLanguage returns "go"
func (r *GoResolver) GetLanguage() string {
	return "go"
}

// ResolveImport resolves a Go import path to source files
func (r *GoResolver) ResolveImport(importPath string, sourceFile string) (*ImportResolution, error) {
	// Handle special cases
	if importPath == "C" {
		return NewUnresolvableResolution(importPath, "go", "cgo pseudo-package"), nil
	}

	// Determine source directory for go list context
	sourceDir := filepath.Dir(sourceFile)

	// Use go list to get package information
	cmd := exec.Command("go", "list", "-json", importPath)
	cmd.Dir = sourceDir
	output, err := cmd.Output()

	if err != nil {
		// Package not found or error
		return NewUnresolvableResolution(importPath, "go", fmt.Sprintf("go list failed: %v", err)), nil
	}

	// Parse go list output
	var pkgInfo goPackageInfo

	if err := json.Unmarshal(output, &pkgInfo); err != nil {
		return NewUnresolvableResolution(importPath, "go", fmt.Sprintf("failed to parse go list output: %v", err)), nil
	}

	// Determine package type
	packageType := r.classifyPackage(&pkgInfo)

	// Get all .go files in the package directory
	if pkgInfo.Dir == "" {
		return NewUnresolvableResolution(importPath, "go", "no package directory found"), nil
	}

	var filePaths []string
	if len(pkgInfo.GoFiles) > 0 {
		// Use the files reported by go list (more accurate)
		for _, file := range pkgInfo.GoFiles {
			fullPath := filepath.Join(pkgInfo.Dir, file)
			filePaths = append(filePaths, fullPath)
		}
	} else {
		// Fallback: glob for .go files
		matches, err := filepath.Glob(filepath.Join(pkgInfo.Dir, "*.go"))
		if err != nil {
			return NewUnresolvableResolution(importPath, "go", fmt.Sprintf("failed to find .go files: %v", err)), nil
		}
		filePaths = matches
	}

	// Filter out test files (*_test.go)
	var nonTestFiles []string
	for _, file := range filePaths {
		if !strings.HasSuffix(file, "_test.go") {
			nonTestFiles = append(nonTestFiles, file)
		}
	}

	if len(nonTestFiles) == 0 {
		return NewUnresolvableResolution(importPath, "go", "no non-test .go files found"), nil
	}

	return NewResolvedResolution(importPath, "go", nonTestFiles, packageType), nil
}

// classifyPackage determines if a package is stdlib, local, or external
func (r *GoResolver) classifyPackage(pkgInfo *goPackageInfo) string {
	// Standard library
	if pkgInfo.Standard {
		return PackageTypeStdlib
	}

	// Local package (same module)
	if r.ModulePath != "" && pkgInfo.Module != nil {
		if pkgInfo.Module.Path == r.ModulePath {
			return PackageTypeLocal
		}
	}

	// External dependency
	return PackageTypeExternal
}

// findGoModulePath finds the module path from go.mod
// Returns empty string if not in a module
func findGoModulePath(sourceFile string) (string, error) {
	dir := filepath.Dir(sourceFile)

	// Walk up directories looking for go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, read module path
			data, err := os.ReadFile(goModPath)
			if err != nil {
				return "", err
			}

			// Parse module line: "module github.com/user/project"
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					modPath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
					return modPath, nil
				}
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found")
}
