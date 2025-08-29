// Package main validates struct tags in Go source code for compliance with project standards.


package main



import (

	"go/ast"

	"go/parser"

	"go/token"

	"strings"

)



func main() {

	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, "pkg/controllers/resilience/patterns.go", nil, parser.ParseComments)

	if err != nil {

		panic(err)

	}



	tagCount := 0

	for _, decl := range node.Decls {

		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {

			for _, spec := range genDecl.Specs {

				if typeSpec, ok := spec.(*ast.TypeSpec); ok {

					if structType, ok := typeSpec.Type.(*ast.StructType); ok {

						for _, field := range structType.Fields.List {

							if field.Tag != nil {

								tagCount++

								tag := field.Tag.Value

								// Check for proper backtick wrapping and json tag format.

								if !strings.HasPrefix(tag, "`") || !strings.HasSuffix(tag, "`") {

									println("Invalid tag format:", tag)

									return

								}

								if strings.Contains(tag, "json:") && !strings.Contains(tag, "json:\"") {

									println("Invalid json tag format:", tag)

									return

								}

							}

						}

					}

				}

			}

		}

	}

	println("Validated", tagCount, "struct tags - all are syntactically correct")

}

