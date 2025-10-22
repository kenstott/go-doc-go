package importer

import (
	"os"
	"path/filepath"
	"strings"
)

// JavaResolver resolves Java import paths to source files
type JavaResolver struct {
	// SourceRoots are potential Java source roots (src/main/java, src/, etc.)
	SourceRoots []string
}

// NewJavaResolver creates a new Java import resolver
// sourceFile should be a .java file to help determine source roots
func NewJavaResolver(sourceFile string) *JavaResolver {
	resolver := &JavaResolver{}
	resolver.SourceRoots = resolver.findSourceRoots(sourceFile)
	return resolver
}

// GetLanguage returns "java"
func (r *JavaResolver) GetLanguage() string {
	return "java"
}

// ResolveImport resolves a Java import to source files
func (r *JavaResolver) ResolveImport(importPath string, sourceFile string) (*ImportResolution, error) {
	// Check if it's a standard library package
	if r.isStdlibPackage(importPath) {
		return NewExcludedResolution(importPath, "java", PackageTypeStdlib, "Java standard library"), nil
	}

	// Convert package name to file path
	// com.example.Foo -> com/example/Foo.java
	filePath := strings.ReplaceAll(importPath, ".", string(os.PathSeparator)) + ".java"

	// Try each source root
	for _, root := range r.SourceRoots {
		fullPath := filepath.Join(root, filePath)
		if _, err := os.Stat(fullPath); err == nil {
			// Found the file
			packageType := r.classifyPackage(importPath, sourceFile, root)
			return NewResolvedResolution(importPath, "java", []string{fullPath}, packageType), nil
		}
	}

	// Not found in source roots - might be a JAR dependency
	return NewUnresolvableResolution(importPath, "java", "source file not found (may be JAR-only dependency)"), nil
}

// findSourceRoots finds potential Java source roots
func (r *JavaResolver) findSourceRoots(sourceFile string) []string {
	var roots []string
	dir := filepath.Dir(sourceFile)

	// Walk up directories looking for common Java source root patterns
	current := dir
	for {
		// Check for Maven structure (src/main/java)
		mavenRoot := filepath.Join(current, "src", "main", "java")
		if _, err := os.Stat(mavenRoot); err == nil {
			roots = append(roots, mavenRoot)
		}

		// Check for Gradle structure (src/main/java)
		gradleRoot := filepath.Join(current, "src", "main", "java")
		if _, err := os.Stat(gradleRoot); err == nil && !contains(roots, gradleRoot) {
			roots = append(roots, gradleRoot)
		}

		// Check for simple src/ root
		srcRoot := filepath.Join(current, "src")
		if _, err := os.Stat(srcRoot); err == nil && !contains(roots, srcRoot) {
			roots = append(roots, srcRoot)
		}

		// Move up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}

	// If no roots found, use the directory containing the source file
	if len(roots) == 0 {
		roots = append(roots, dir)
	}

	return roots
}

// isStdlibPackage checks if a package is part of the Java standard library
func (r *JavaResolver) isStdlibPackage(pkg string) bool {
	// Java standard library packages
	stdlibPrefixes := []string{
		"java.", "javax.", "org.xml.", "org.w3c.", "org.omg.", "org.ietf.",
	}

	for _, prefix := range stdlibPrefixes {
		if strings.HasPrefix(pkg, prefix) {
			return true
		}
	}

	return false
}

// classifyPackage determines if a package is local or external
func (r *JavaResolver) classifyPackage(importPath string, sourceFile string, sourceRoot string) string {
	// Extract package name from source file
	sourcePackage := r.getPackageFromFile(sourceFile, sourceRoot)

	// If import is in the same package tree, it's local
	if sourcePackage != "" && strings.HasPrefix(importPath, sourcePackage) {
		return PackageTypeLocal
	}

	// Check if the import is under the same source root (same project)
	importFilePath := strings.ReplaceAll(importPath, ".", string(os.PathSeparator))
	fullImportPath := filepath.Join(sourceRoot, importFilePath)

	sourceFileDir := filepath.Dir(sourceFile)
	if strings.HasPrefix(fullImportPath, sourceFileDir) || strings.HasPrefix(sourceFileDir, filepath.Dir(fullImportPath)) {
		return PackageTypeLocal
	}

	// Otherwise it's external
	return PackageTypeExternal
}

// getPackageFromFile extracts the package name from a Java file's path
func (r *JavaResolver) getPackageFromFile(filePath string, sourceRoot string) string {
	// Get relative path from source root
	relPath, err := filepath.Rel(sourceRoot, filepath.Dir(filePath))
	if err != nil {
		return ""
	}

	// Convert file path to package name
	pkg := strings.ReplaceAll(relPath, string(os.PathSeparator), ".")
	if pkg == "." {
		return ""
	}

	return pkg
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
