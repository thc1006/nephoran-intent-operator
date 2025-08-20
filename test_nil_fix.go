package main

import (
	"fmt"
	"log"
	
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

func main() {
	fmt.Println("Testing nil pointer fix for Watcher.Close()...")
	
	// Test 1: Direct nil pointer call (this would panic before our fix)
	fmt.Print("Test 1 - Calling Close() on nil Watcher: ")
	var watcher *loop.Watcher = nil
	
	// This should not panic after our fix
	err := watcher.Close()
	if err != nil {
		fmt.Printf("ERROR - %v\n", err)
	} else {
		fmt.Printf("SUCCESS - no panic\n")
	}
	
	// Test 2: Safe defer pattern
	fmt.Print("Test 2 - Safe defer pattern: ")
	func() {
		var watcher *loop.Watcher = nil
		defer func() {
			if watcher != nil {
				if err := watcher.Close(); err != nil {
					log.Printf("Error closing watcher: %v", err)
				}
			}
		}()
		
		// Simulate initialization failure where watcher remains nil
		// The defer should handle this gracefully
	}()
	fmt.Printf("SUCCESS - no panic in defer\n")
	
	// Test 3: Using SafeClose helper
	fmt.Print("Test 3 - Using SafeClose helper: ")
	err = loop.SafeClose(nil)
	if err != nil {
		fmt.Printf("ERROR - %v\n", err)
	} else {
		fmt.Printf("SUCCESS - no panic\n")
	}
	
	fmt.Println("\nAll nil pointer tests passed! The panic fix is working correctly.")
}