package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("=== O2 Providers Package Analysis ===")

	providersPath := "pkg/oran/o2/providers"

	// Check if directory exists
	if _, err := os.Stat(providersPath); os.IsNotExist(err) {
		fmt.Printf("❌ Directory %s does not exist\n", providersPath)

		// Check parent directories
		checkDir("pkg")
		checkDir("pkg/oran")
		checkDir("pkg/oran/o2")

		fmt.Println("\n=== Creating missing providers directory ===")
		os.MkdirAll(providersPath, 0755)
		return
	}

	fmt.Printf("✅ Directory %s exists\n", providersPath)

	// Walk through and analyze Go files
	err := filepath.Walk(providersPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fmt.Printf("\n📄 Analyzing: %s\n", path)
		analyzeGoFile(path)
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
}

func checkDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("❌ %s does not exist\n", path)
	} else {
		fmt.Printf("✅ %s exists\n", path)
	}
}

func analyzeGoFile(filename string) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("  ❌ Parse error: %v\n", err)
		return
	}

	fmt.Printf("  📦 Package: %s\n", node.Name.Name)

	// Check imports
	for _, imp := range node.Imports {
		fmt.Printf("  📥 Import: %s\n", imp.Path.Value)
	}

	// Check types
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			fmt.Printf("  🏷️  Type: %s\n", x.Name.Name)
		case *ast.FuncDecl:
			if x.Name.IsExported() {
				fmt.Printf("  🔧 Function: %s\n", x.Name.Name)
			}
		}
		return true
	})
}
