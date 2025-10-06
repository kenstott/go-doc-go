package main

import (
	"fmt"
	"log"

	ort "github.com/yalue/onnxruntime_go"
)

func main() {
	// Initialize ONNX Runtime
	libraryPath := "/Users/kennethstott/homebrew/lib/libonnxruntime.dylib"
	ort.SetSharedLibraryPath(libraryPath)

	if err := ort.InitializeEnvironment(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	defer ort.DestroyEnvironment()

	// Try to create session with CoreML
	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		log.Fatalf("Failed to create session options: %v", err)
	}
	defer sessionOptions.Destroy()

	// Try CoreML
	fmt.Println("Attempting to append CoreML execution provider...")
	if err := sessionOptions.AppendExecutionProviderCoreML(0); err != nil {
		fmt.Printf("❌ CoreML NOT available: %v\n", err)
		fmt.Println("✅ Falling back to CPU execution provider")
	} else {
		fmt.Println("✅ CoreML execution provider successfully added!")
		fmt.Println("   GPU/Neural Engine acceleration will be used")
	}

	fmt.Println("\nConclusion:")
	if err != nil {
		fmt.Println("  - ONNX will use CPU-only inference")
		fmt.Println("  - This is slower but still functional")
	} else {
		fmt.Println("  - ONNX will use Apple Silicon GPU/Neural Engine")
		fmt.Println("  - This provides significant acceleration")
	}
}
