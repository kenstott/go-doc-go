package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// JSResolver resolves JavaScript/TypeScript import paths to source files
type JSResolver struct {
	// NodeBuiltins is a set of Node.js built-in modules
	NodeBuiltins map[string]bool
}

// NewJSResolver creates a new JavaScript/TypeScript import resolver
func NewJSResolver() *JSResolver {
	// Node.js built-in modules (common ones)
	builtins := map[string]bool{
		"assert": true, "buffer": true, "child_process": true, "cluster": true,
		"crypto": true, "dgram": true, "dns": true, "domain": true,
		"events": true, "fs": true, "http": true, "https": true,
		"net": true, "os": true, "path": true, "punycode": true,
		"querystring": true, "readline": true, "stream": true, "string_decoder": true,
		"timers": true, "tls": true, "tty": true, "url": true,
		"util": true, "v8": true, "vm": true, "zlib": true,
		// Node.js prefixed versions
		"node:assert": true, "node:buffer": true, "node:child_process": true,
		"node:crypto": true, "node:fs": true, "node:http": true, "node:path": true,
		"node:stream": true, "node:url": true, "node:util": true,
	}

	return &JSResolver{
		NodeBuiltins: builtins,
	}
}

// GetLanguage returns "javascript"
func (r *JSResolver) GetLanguage() string {
	return "javascript"
}

// ResolveImport resolves a JavaScript/TypeScript import to source files
func (r *JSResolver) ResolveImport(importPath string, sourceFile string) (*ImportResolution, error) {
	// Check if it's a Node.js built-in module
	if r.NodeBuiltins[importPath] {
		return NewExcludedResolution(importPath, "javascript", PackageTypeBuiltin, "Node.js built-in module"), nil
	}

	sourceDir := filepath.Dir(sourceFile)

	// Handle relative imports (./foo, ../bar)
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		resolved := r.resolveRelativeImport(importPath, sourceDir)
		if resolved != nil {
			return resolved, nil
		}
		return NewUnresolvableResolution(importPath, "javascript", "relative import not found"), nil
	}

	// Handle package imports (from node_modules)
	resolved := r.resolvePackageImport(importPath, sourceDir)
	if resolved != nil {
		return resolved, nil
	}

	return NewUnresolvableResolution(importPath, "javascript", "package not found in node_modules"), nil
}

// resolveRelativeImport resolves ./foo or ../bar style imports
func (r *JSResolver) resolveRelativeImport(importPath string, sourceDir string) *ImportResolution {
	// Resolve relative to source directory
	basePath := filepath.Join(sourceDir, importPath)

	// Try different extensions
	extensions := []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}

	// First try exact path with extensions
	for _, ext := range extensions {
		filePath := basePath + ext
		if _, err := os.Stat(filePath); err == nil {
			return NewResolvedResolution(importPath, "javascript", []string{filePath}, PackageTypeLocal)
		}
	}

	// Try as directory with index file
	for _, ext := range extensions {
		indexPath := filepath.Join(basePath, "index"+ext)
		if _, err := os.Stat(indexPath); err == nil {
			return NewResolvedResolution(importPath, "javascript", []string{indexPath}, PackageTypeLocal)
		}
	}

	return nil
}

// resolvePackageImport resolves package imports from node_modules
func (r *JSResolver) resolvePackageImport(importPath string, sourceDir string) *ImportResolution {
	// Handle scoped packages (@org/package/sub/path)
	var packageName string
	var subPath string

	if strings.HasPrefix(importPath, "@") {
		// Scoped package: @org/package or @org/package/sub/path
		parts := strings.SplitN(importPath, "/", 3)
		if len(parts) >= 2 {
			packageName = parts[0] + "/" + parts[1] // @org/package
			if len(parts) == 3 {
				subPath = parts[2] // sub/path
			}
		}
	} else {
		// Regular package: package or package/sub/path
		parts := strings.SplitN(importPath, "/", 2)
		packageName = parts[0]
		if len(parts) == 2 {
			subPath = parts[1]
		}
	}

	// Walk up directories looking for node_modules
	dir := sourceDir
	for {
		nodeModulesPath := filepath.Join(dir, "node_modules", packageName)
		if _, err := os.Stat(nodeModulesPath); err == nil {
			// Found the package
			var resolvedPath string
			if subPath != "" {
				// Importing sub-path within package
				resolvedPath = filepath.Join(nodeModulesPath, subPath)
			} else {
				// Importing package root - find entry point
				entryPoint := r.findPackageEntryPoint(nodeModulesPath)
				if entryPoint == "" {
					return NewUnresolvableResolution(importPath, "javascript", "no entry point found in package")
				}
				resolvedPath = entryPoint
			}

			// Try resolving as file or directory
			files := r.resolvePathToFiles(resolvedPath)
			if len(files) > 0 {
				return NewResolvedResolution(importPath, "javascript", files, PackageTypeExternal)
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

	return nil
}

// findPackageEntryPoint reads package.json to find the entry point
func (r *JSResolver) findPackageEntryPoint(packageDir string) string {
	packageJSONPath := filepath.Join(packageDir, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		// No package.json, try index.js
		return filepath.Join(packageDir, "index.js")
	}

	var pkg struct {
		Main   string            `json:"main"`
		Exports interface{}       `json:"exports"`
		Types  string            `json:"types"`
		Module string            `json:"module"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return filepath.Join(packageDir, "index.js")
	}

	// Prefer TypeScript types if available
	if pkg.Types != "" {
		return filepath.Join(packageDir, pkg.Types)
	}

	// Then ESM module entry
	if pkg.Module != "" {
		return filepath.Join(packageDir, pkg.Module)
	}

	// Then main entry
	if pkg.Main != "" {
		return filepath.Join(packageDir, pkg.Main)
	}

	// Default to index.js
	return filepath.Join(packageDir, "index.js")
}

// resolvePathToFiles resolves a path to actual source files
func (r *JSResolver) resolvePathToFiles(path string) []string {
	// Try as file with extensions
	extensions := []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}

	// First try exact path
	if _, err := os.Stat(path); err == nil {
		// Could be a file or directory
		info, _ := os.Stat(path)
		if !info.IsDir() {
			return []string{path}
		}
		// It's a directory, try index files
		for _, ext := range extensions {
			indexPath := filepath.Join(path, "index"+ext)
			if _, err := os.Stat(indexPath); err == nil {
				return []string{indexPath}
			}
		}
		return nil
	}

	// Try adding extensions
	for _, ext := range extensions {
		filePath := path + ext
		if _, err := os.Stat(filePath); err == nil {
			return []string{filePath}
		}
	}

	return nil
}
