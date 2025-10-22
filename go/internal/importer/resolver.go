package importer

// ImportResolver defines the interface for resolving imports to file paths
// Different languages implement this interface with language-specific resolution logic
type ImportResolver interface {
	// ResolveImport attempts to resolve an import path to actual file paths
	// importPath: the import/dependency string (e.g., "fmt", "github.com/user/pkg", "./utils")
	// sourceFile: the absolute path of the file containing the import (for relative resolution)
	// Returns resolution result with status and file paths
	ResolveImport(importPath string, sourceFile string) (*ImportResolution, error)

	// GetLanguage returns the language this resolver handles (go, javascript, python, java)
	GetLanguage() string
}

// ImportResolution represents the result of import resolution
type ImportResolution struct {
	// Status indicates the resolution result: "resolved", "unresolvable", or "excluded"
	Status string

	// FilePaths contains absolute paths to source files for this import
	// Empty if Status is not "resolved"
	FilePaths []string

	// PackageType classifies the import: "stdlib", "local", "external", or "builtin"
	PackageType string

	// Reason explains why resolution failed or was excluded (for Status != "resolved")
	Reason string

	// ImportPath is the original import path that was resolved
	ImportPath string

	// Language is the programming language
	Language string
}

// Status constants
const (
	StatusResolved    = "resolved"     // Successfully resolved to file path(s)
	StatusUnresolvable = "unresolvable" // Cannot be resolved (missing, error, etc.)
	StatusExcluded    = "excluded"     // Resolved but excluded by filters
)

// PackageType constants
const (
	PackageTypeStdlib   = "stdlib"   // Language standard library
	PackageTypeLocal    = "local"    // Same project/module/package
	PackageTypeExternal = "external" // Third-party dependency
	PackageTypeBuiltin  = "builtin"  // Built-in to runtime (e.g., Node.js builtins)
)

// NewUnresolvableResolution creates an unresolvable resolution with a reason
func NewUnresolvableResolution(importPath, language, reason string) *ImportResolution {
	return &ImportResolution{
		Status:     StatusUnresolvable,
		FilePaths:  []string{},
		Reason:     reason,
		ImportPath: importPath,
		Language:   language,
	}
}

// NewExcludedResolution creates an excluded resolution with package type and reason
func NewExcludedResolution(importPath, language, packageType, reason string) *ImportResolution {
	return &ImportResolution{
		Status:      StatusExcluded,
		FilePaths:   []string{},
		PackageType: packageType,
		Reason:      reason,
		ImportPath:  importPath,
		Language:    language,
	}
}

// NewResolvedResolution creates a resolved resolution with file paths and package type
func NewResolvedResolution(importPath, language string, filePaths []string, packageType string) *ImportResolution {
	return &ImportResolution{
		Status:      StatusResolved,
		FilePaths:   filePaths,
		PackageType: packageType,
		ImportPath:  importPath,
		Language:    language,
	}
}
