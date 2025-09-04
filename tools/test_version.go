//go:build ignore
// +build ignore

package main

import "fmt"

// Version information set by ldflags during build
var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	fmt.Printf("Bitcoin Sprint Version: %s\n", Version)
	fmt.Printf("Git Commit: %s\n", Commit)
}

func main() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit: %s\n", Commit)
}
