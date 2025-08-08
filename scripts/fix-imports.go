package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run fix-imports.go <file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inImportBlock := false
	
	for _, line := range lines {
		// Check if we're entering import block
		if strings.TrimSpace(line) == "import (" {
			inImportBlock = true
			newLines = append(newLines, line)
			continue
		}
		
		// Check if we're leaving import block
		if inImportBlock && strings.TrimSpace(line) == ")" {
			inImportBlock = false
			newLines = append(newLines, line)
			continue
		}
		
		// If we're in import block, filter out malformed lines
		if inImportBlock {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "/d") || strings.Contains(trimmed, "lient-go") || strings.Contains(trimmed, "ontroller-runtime") {
				// Skip malformed lines
				continue
			}
		}
		
		newLines = append(newLines, line)
	}
	
	// Write the fixed content back
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(filename, []byte(newContent), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Fixed imports in %s\n", filename)
}