package importer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonResolver resolves Python import paths to source files
type PythonResolver struct {
	// StdlibModules is a set of Python standard library modules
	StdlibModules map[string]bool
}

// NewPythonResolver creates a new Python import resolver
func NewPythonResolver() *PythonResolver {
	// Common Python standard library modules
	stdlib := map[string]bool{
		"abc": true, "argparse": true, "ast": true, "asyncio": true,
		"base64": true, "collections": true, "concurrent": true, "copy": true,
		"csv": true, "datetime": true, "decimal": true, "email": true,
		"enum": true, "functools": true, "glob": true, "hashlib": true,
		"html": true, "http": true, "importlib": true, "io": true,
		"itertools": true, "json": true, "logging": true, "math": true,
		"multiprocessing": true, "operator": true, "os": true, "pathlib": true,
		"pickle": true, "platform": true, "pprint": true, "queue": true,
		"random": true, "re": true, "shutil": true, "signal": true,
		"socket": true, "sqlite3": true, "ssl": true, "string": true,
		"struct": true, "subprocess": true, "sys": true, "tempfile": true,
		"threading": true, "time": true, "traceback": true, "typing": true,
		"unittest": true, "urllib": true, "uuid": true, "warnings": true,
		"weakref": true, "xml": true, "zipfile": true,
	}

	return &PythonResolver{
		StdlibModules: stdlib,
	}
}

// GetLanguage returns "python"
func (r *PythonResolver) GetLanguage() string {
	return "python"
}

// ResolveImport resolves a Python import to source files
func (r *PythonResolver) ResolveImport(importPath string, sourceFile string) (*ImportResolution, error) {
	// Get the top-level module name
	topLevel := strings.Split(importPath, ".")[0]

	// Check if it's a standard library module
	if r.StdlibModules[topLevel] {
		return NewExcludedResolution(importPath, "python", PackageTypeStdlib, "Python standard library"), nil
	}

	sourceDir := filepath.Dir(sourceFile)

	// Handle relative imports (starting with .)
	if strings.HasPrefix(importPath, ".") {
		resolved := r.resolveRelativeImport(importPath, sourceFile, sourceDir)
		if resolved != nil {
			return resolved, nil
		}
		return NewUnresolvableResolution(importPath, "python", "relative import not found"), nil
	}

	// Try to resolve as local package (same project)
	resolved := r.resolveLocalPackage(importPath, sourceDir)
	if resolved != nil {
		return resolved, nil
	}

	// Try to resolve as installed package (site-packages)
	resolved = r.resolveInstalledPackage(importPath)
	if resolved != nil {
		return resolved, nil
	}

	return NewUnresolvableResolution(importPath, "python", "module not found"), nil
}

// resolveRelativeImport resolves relative imports (from . import foo, from ..pkg import bar)
func (r *PythonResolver) resolveRelativeImport(importPath string, sourceFile string, sourceDir string) *ImportResolution {
	// Count leading dots to determine parent directory level
	dots := 0
	for i, ch := range importPath {
		if ch == '.' {
			dots++
		} else {
			importPath = importPath[i:]
			break
		}
	}

	// Move up 'dots-1' directories (one dot means current package)
	targetDir := sourceDir
	for i := 1; i < dots; i++ {
		targetDir = filepath.Dir(targetDir)
	}

	// If there's a module name after the dots, resolve it
	if importPath != "" {
		modulePath := strings.ReplaceAll(importPath, ".", string(os.PathSeparator))
		targetPath := filepath.Join(targetDir, modulePath)

		files := r.resolvePathToFiles(targetPath)
		if len(files) > 0 {
			return NewResolvedResolution(importPath, "python", files, PackageTypeLocal)
		}
	} else {
		// Import from current package (__init__.py)
		initPath := filepath.Join(targetDir, "__init__.py")
		if _, err := os.Stat(initPath); err == nil {
			return NewResolvedResolution(importPath, "python", []string{initPath}, PackageTypeLocal)
		}
	}

	return nil
}

// resolveLocalPackage resolves imports as local packages (same project)
func (r *PythonResolver) resolveLocalPackage(importPath string, sourceDir string) *ImportResolution {
	// Convert dotted import to file path
	modulePath := strings.ReplaceAll(importPath, ".", string(os.PathSeparator))

	// Walk up directories looking for the module
	dir := sourceDir
	for {
		targetPath := filepath.Join(dir, modulePath)
		files := r.resolvePathToFiles(targetPath)
		if len(files) > 0 {
			return NewResolvedResolution(importPath, "python", files, PackageTypeLocal)
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}

		// Stop if we hit a directory without __init__.py (not a package)
		initPath := filepath.Join(parent, "__init__.py")
		if _, err := os.Stat(initPath); os.IsNotExist(err) {
			break
		}

		dir = parent
	}

	return nil
}

// resolveInstalledPackage resolves imports from site-packages
func (r *PythonResolver) resolveInstalledPackage(importPath string) *ImportResolution {
	// Use Python to find the module location
	// This is more reliable than trying to guess site-packages location
	topLevel := strings.Split(importPath, ".")[0]

	cmd := exec.Command("python3", "-c", fmt.Sprintf("import %s; print(%s.__file__)", topLevel, topLevel))
	output, err := cmd.Output()
	if err != nil {
		// Module not installed or import failed
		return nil
	}

	modulePath := strings.TrimSpace(string(output))
	if modulePath == "" {
		return nil
	}

	// Check if it's a compiled module (.so, .pyd)
	ext := filepath.Ext(modulePath)
	if ext == ".so" || ext == ".pyd" {
		return NewUnresolvableResolution(importPath, "python", "compiled C extension module")
	}

	// For package imports (a.b.c), try to find the specific submodule
	if strings.Contains(importPath, ".") {
		parts := strings.Split(importPath, ".")
		packageDir := filepath.Dir(modulePath) // Get package directory
		subPath := filepath.Join(parts[1:]...)
		targetPath := filepath.Join(packageDir, subPath)

		files := r.resolvePathToFiles(targetPath)
		if len(files) > 0 {
			return NewResolvedResolution(importPath, "python", files, PackageTypeExternal)
		}
	}

	// Return the top-level module file
	if _, err := os.Stat(modulePath); err == nil {
		return NewResolvedResolution(importPath, "python", []string{modulePath}, PackageTypeExternal)
	}

	return nil
}

// resolvePathToFiles resolves a path to Python source files
func (r *PythonResolver) resolvePathToFiles(path string) []string {
	// Try as file (.py)
	pyPath := path + ".py"
	if _, err := os.Stat(pyPath); err == nil {
		return []string{pyPath}
	}

	// Try as package directory (__init__.py)
	initPath := filepath.Join(path, "__init__.py")
	if _, err := os.Stat(initPath); err == nil {
		return []string{initPath}
	}

	// Try the path itself if it's a .py file
	if strings.HasSuffix(path, ".py") {
		if _, err := os.Stat(path); err == nil {
			return []string{path}
		}
	}

	return nil
}
