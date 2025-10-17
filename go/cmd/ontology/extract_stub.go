package main

import (
	"fmt"
	"os"
	"os/exec"
)

// runExtract wraps the existing ontology_extract binary
func runExtractOld(args []string) {
	// Find ontology_extract binary
	binaryPath := "../bin/ontology_extract"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		binaryPath = "bin/ontology_extract"
	}

	// Forward to existing binary
	cmd := exec.Command(binaryPath, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running extract command: %v\n", err)
		os.Exit(1)
	}
}
