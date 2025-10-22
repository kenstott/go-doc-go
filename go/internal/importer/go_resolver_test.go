package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoResolver_ResolveImport_Stdlib(t *testing.T) {
	// Create a resolver for a test file
	testFile := "/tmp/test.go"
	resolver, err := NewGoResolver(testFile)
	if err != nil {
		t.Skipf("Could not create Go resolver: %v", err)
	}

	// Test standard library import
	resolution, err := resolver.ResolveImport("fmt", testFile)
	if err != nil {
		t.Fatalf("ResolveImport failed: %v", err)
	}

	if resolution.Status != StatusResolved {
		t.Errorf("Expected status %s, got %s", StatusResolved, resolution.Status)
	}

	if resolution.PackageType != PackageTypeStdlib {
		t.Errorf("Expected package type %s, got %s", PackageTypeStdlib, resolution.PackageType)
	}

	if len(resolution.FilePaths) == 0 {
		t.Error("Expected at least one file path for fmt package")
	}

	// Verify files exist and are .go files
	for _, path := range resolution.FilePaths {
		if !strings.HasSuffix(path, ".go") {
			t.Errorf("Expected .go file, got %s", path)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File does not exist: %s", path)
		}
	}
}

func TestGoResolver_ResolveImport_Local(t *testing.T) {
	// Get the path to this test file
	testFile, err := filepath.Abs("go_resolver_test.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	resolver, err := NewGoResolver(testFile)
	if err != nil {
		t.Skipf("Could not create Go resolver: %v", err)
	}

	// Test importing the resolver package itself (local package)
	resolution, err := resolver.ResolveImport("github.com/kennethstott/doculyzer-go-conversion/internal/importer", testFile)
	if err != nil {
		t.Fatalf("ResolveImport failed: %v", err)
	}

	if resolution.Status != StatusResolved {
		t.Errorf("Expected status %s, got %s (reason: %s)", StatusResolved, resolution.Status, resolution.Reason)
	}

	if resolution.PackageType != PackageTypeLocal {
		t.Errorf("Expected package type %s, got %s", PackageTypeLocal, resolution.PackageType)
	}

	if len(resolution.FilePaths) == 0 {
		t.Error("Expected at least one file path for local package")
	}

	// Verify at least one file is in this directory
	foundLocalFile := false
	for _, path := range resolution.FilePaths {
		if strings.Contains(path, "importer") {
			foundLocalFile = true
			break
		}
	}

	if !foundLocalFile {
		t.Errorf("Expected to find files in importer directory, got: %v", resolution.FilePaths)
	}
}

func TestGoResolver_ResolveImport_Unresolvable(t *testing.T) {
	testFile := "/tmp/test.go"
	resolver, err := NewGoResolver(testFile)
	if err != nil {
		t.Skipf("Could not create Go resolver: %v", err)
	}

	// Test non-existent package
	resolution, err := resolver.ResolveImport("github.com/nonexistent/fake/package", testFile)
	if err != nil {
		t.Fatalf("ResolveImport failed: %v", err)
	}

	if resolution.Status != StatusUnresolvable {
		t.Errorf("Expected status %s, got %s", StatusUnresolvable, resolution.Status)
	}

	if resolution.Reason == "" {
		t.Error("Expected a reason for unresolvable import")
	}
}

func TestGoResolver_ResolveImport_CgoPseudoPackage(t *testing.T) {
	testFile := "/tmp/test.go"
	resolver, err := NewGoResolver(testFile)
	if err != nil {
		t.Skipf("Could not create Go resolver: %v", err)
	}

	// Test cgo pseudo-package
	resolution, err := resolver.ResolveImport("C", testFile)
	if err != nil {
		t.Fatalf("ResolveImport failed: %v", err)
	}

	if resolution.Status != StatusUnresolvable {
		t.Errorf("Expected status %s for cgo, got %s", StatusUnresolvable, resolution.Status)
	}

	if !strings.Contains(resolution.Reason, "cgo") {
		t.Errorf("Expected reason to mention cgo, got: %s", resolution.Reason)
	}
}

func TestGoResolver_FindGoModulePath(t *testing.T) {
	// Get the path to this test file
	testFile, err := filepath.Abs("go_resolver_test.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	modulePath, err := findGoModulePath(testFile)
	if err != nil {
		t.Fatalf("findGoModulePath failed: %v", err)
	}

	expectedPath := "github.com/kennethstott/doculyzer-go-conversion"
	if modulePath != expectedPath {
		t.Errorf("Expected module path %s, got %s", expectedPath, modulePath)
	}
}
