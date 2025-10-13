package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kennethstott/doculyzer-go-conversion/internal/cache"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands: get <key>, set <key> <value>, clear, size\n")
		os.Exit(1)
	}

	// Parse flags for cache configuration
	var maxSize int
	var ttl int
	var cacheFile string
	var useMmap bool

	// Create a custom flag set for parsing after the command
	flagSet := flag.NewFlagSet("cache", flag.ExitOnError)
	flagSet.IntVar(&maxSize, "max-size", 128, "Maximum cache size")
	flagSet.IntVar(&ttl, "ttl", 3600, "TTL in seconds")
	flagSet.StringVar(&cacheFile, "cache-file", "", "Cache persistence file")
	flagSet.BoolVar(&useMmap, "use-mmap", true, "Use memory-mapped file for better performance")

	// Parse flags starting from position 2 (after command)
	flagArgs := []string{}
	nonFlagArgs := []string{os.Args[1]} // Keep the command

	for i := 2; i < len(os.Args); i++ {
		if len(os.Args[i]) > 0 && os.Args[i][0] == '-' {
			// This is a flag
			flagArgs = append(flagArgs, os.Args[i])
			// Check if next arg is the flag value
			if i+1 < len(os.Args) && os.Args[i+1][0] != '-' {
				flagArgs = append(flagArgs, os.Args[i+1])
				i++
			}
		} else {
			// This is a non-flag argument
			nonFlagArgs = append(nonFlagArgs, os.Args[i])
		}
	}

	flagSet.Parse(flagArgs)

	// Default cache file location if not specified
	if cacheFile == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir := filepath.Join(homeDir, ".go-doc-go", "cache")
		os.MkdirAll(cacheDir, 0755)

		if useMmap {
			cacheFile = filepath.Join(cacheDir, fmt.Sprintf("lru_mmap_%d_%d.bin", maxSize, ttl))
		} else {
			cacheFile = filepath.Join(cacheDir, fmt.Sprintf("lru_%d_%d.json", maxSize, ttl))
		}
	}

	command := nonFlagArgs[0]

	// Use memory-mapped cache for better performance
	if useMmap {
		mmapCache, err := cache.NewMmapCache(cacheFile, maxSize, ttl, 1024*1024) // 1MB file
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing mmap cache: %v\n", err)
			os.Exit(1)
		}
		defer mmapCache.Close()

		switch command {
		case "get":
			if len(nonFlagArgs) < 2 {
				fmt.Fprintf(os.Stderr, "Usage: %s get <key>\n", os.Args[0])
				os.Exit(1)
			}
			handleMmapGet(mmapCache, nonFlagArgs[1])

		case "set":
			if len(nonFlagArgs) < 3 {
				// Check if we have a key but empty value
				if len(nonFlagArgs) == 2 {
					// Set empty string value
					handleMmapSet(mmapCache, nonFlagArgs[1], "")
				} else {
					fmt.Fprintf(os.Stderr, "Usage: %s set <key> <value>\n", os.Args[0])
					os.Exit(1)
				}
			} else {
				handleMmapSet(mmapCache, nonFlagArgs[1], nonFlagArgs[2])
			}

		case "clear":
			handleMmapClear(mmapCache)

		case "size":
			handleMmapSize(mmapCache)

		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
			os.Exit(1)
		}
	} else {
		// Fall back to simpler cache implementation if needed
		fmt.Fprintf(os.Stderr, "Non-mmap mode not implemented in this version\n")
		os.Exit(1)
	}
}

func handleMmapGet(mc *cache.MmapCache, key string) {
	value, found := mc.Get(key)
	if !found {
		fmt.Println("null")
		os.Exit(0)
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling value: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

func handleMmapSet(mc *cache.MmapCache, key string, valueStr string) {
	// Try to parse as JSON first
	var value interface{}
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		// If not valid JSON, treat as string
		value = valueStr
	}

	if err := mc.Set(key, value); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting value: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK")
}

func handleMmapClear(mc *cache.MmapCache) {
	if err := mc.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
}

func handleMmapSize(mc *cache.MmapCache) {
	size := mc.Size()
	fmt.Println(strconv.Itoa(size))
}