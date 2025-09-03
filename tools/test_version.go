package main

import "fmt"

// Version information set by ldflags during build
var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit: %s\n", Commit)
}
