package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	ort "github.com/yalue/onnxruntime_go"
)

func main() {
	// Set library path to CoreML-enabled version
	homeDir, _ := os.UserHomeDir()
	venvPath := filepath.Join(homeDir, "PycharmProjects", "doculyzer-go-conversion", ".venv", "lib", "python3.12", "site-packages", "onnxruntime", "capi", "libonnxruntime.1.23.0.dylib")

	log.Printf("Setting library path to: %s", venvPath)
	ort.SetSharedLibraryPath(venvPath)

	// Initialize ONNX Runtime
	err := ort.InitializeEnvironment()
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	defer ort.DestroyEnvironment()

	log.Println("ONNX Runtime initialized successfully")

	// Try to create a session with CoreML
	options, err := ort.NewSessionOptions()
	if err != nil {
		log.Fatalf("Failed to create session options: %v", err)
	}
	defer options.Destroy()

	log.Println("Attempting to enable CoreML execution provider...")
	err = options.AppendExecutionProviderCoreML(0)
	if err != nil {
		log.Printf("CoreML NOT available: %v", err)
		fmt.Println("RESULT: CoreML support is NOT available")
		os.Exit(1)
	}

	log.Println("SUCCESS: CoreML execution provider enabled!")
	fmt.Println("RESULT: CoreML support IS available and working!")
}
