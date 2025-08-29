
package main



import (

	"bufio"

	"fmt"

	"go/ast"

	"go/format"

	"go/parser"

	"go/token"

	"log"

	"os"

	"path/filepath"

	"regexp"

	"strings"

)



// LintFixer applies common golangci-lint fixes for 2025 best practices.

type LintFixer struct {

	fileSet *token.FileSet

	fixes   map[string]func(*ast.File) bool

}



// NewLintFixer creates a new lint fixer with 2025 patterns.

func NewLintFixer() *LintFixer {

	fixer := &LintFixer{

		fileSet: token.NewFileSet(),

		fixes:   make(map[string]func(*ast.File) bool),

	}



	// Register fix functions.

	fixer.fixes["package-comment"] = fixer.addPackageComment

	fixer.fixes["exported-docs"] = fixer.addExportedDocs

	fixer.fixes["error-wrapping"] = fixer.fixErrorWrapping

	fixer.fixes["unused-params"] = fixer.fixUnusedParams

	fixer.fixes["context-first"] = fixer.ensureContextFirst



	return fixer

}



// FixFile applies all registered fixes to a Go file.

func (lf *LintFixer) FixFile(filename string) error {

	// Parse the Go file.

	src, err := os.ReadFile(filename)

	if err != nil {

		return fmt.Errorf("failed to read file %s: %w", filename, err)

	}



	file, err := parser.ParseFile(lf.fileSet, filename, src, parser.ParseComments)

	if err != nil {

		return fmt.Errorf("failed to parse file %s: %w", filename, err)

	}



	// Apply fixes.

	modified := false

	for fixName, fixFunc := range lf.fixes {

		if fixFunc(file) {

			modified = true

			fmt.Printf("Applied fix '%s' to %s\n", fixName, filename)

		}

	}



	// Write back if modified.

	if modified {

		var buf strings.Builder

		if err := format.Node(&buf, lf.fileSet, file); err != nil {

			return fmt.Errorf("failed to format file %s: %w", filename, err)

		}



		if err := os.WriteFile(filename, []byte(buf.String()), 0o640); err != nil {

			return fmt.Errorf("failed to write file %s: %w", filename, err)

		}



		fmt.Printf("Updated %s\n", filename)

	}



	return nil

}



// addPackageComment adds missing package comments.

func (lf *LintFixer) addPackageComment(file *ast.File) bool {

	// Check if package already has a comment.

	if file.Doc != nil && len(file.Doc.List) > 0 {

		return false

	}



	// Generate package comment based on package name.

	packageName := file.Name.Name

	comment := generatePackageComment(packageName)



	// Create comment group.

	commentGroup := &ast.CommentGroup{

		List: []*ast.Comment{

			{Text: comment},

		},

	}



	file.Doc = commentGroup

	return true

}



// generatePackageComment generates appropriate package comment based on name.

func generatePackageComment(packageName string) string {

	switch {

	case strings.Contains(packageName, "client"):

		return fmt.Sprintf("// Package %s provides client implementations for service communication.", packageName)

	case strings.Contains(packageName, "auth"):

		return fmt.Sprintf("// Package %s provides authentication and authorization functionality.", packageName)

	case strings.Contains(packageName, "config"):

		return fmt.Sprintf("// Package %s provides configuration management and validation.", packageName)

	case strings.Contains(packageName, "controller"):

		return fmt.Sprintf("// Package %s provides Kubernetes controller implementations.", packageName)

	case strings.Contains(packageName, "webhook"):

		return fmt.Sprintf("// Package %s provides Kubernetes webhook implementations.", packageName)

	case strings.Contains(packageName, "security"):

		return fmt.Sprintf("// Package %s provides security utilities and mTLS functionality.", packageName)

	case strings.Contains(packageName, "monitoring"):

		return fmt.Sprintf("// Package %s provides monitoring and observability features.", packageName)

	case strings.Contains(packageName, "audit"):

		return fmt.Sprintf("// Package %s provides audit logging and compliance tracking.", packageName)

	case packageName == "main":

		return "// Package main provides the main entry point for the application."

	default:

		return fmt.Sprintf("// Package %s provides core functionality for the Nephoran Intent Operator.", packageName)

	}

}



// addExportedDocs adds missing documentation for exported functions and types.

func (lf *LintFixer) addExportedDocs(file *ast.File) bool {

	modified := false



	ast.Inspect(file, func(n ast.Node) bool {

		switch node := n.(type) {

		case *ast.FuncDecl:

			if node.Name.IsExported() && node.Doc == nil {

				node.Doc = generateFunctionDoc(node)

				modified = true

			}

		case *ast.GenDecl:

			if node.Tok == token.TYPE {

				for _, spec := range node.Specs {

					if typeSpec, ok := spec.(*ast.TypeSpec); ok {

						if typeSpec.Name.IsExported() && typeSpec.Doc == nil && node.Doc == nil {

							node.Doc = generateTypeDoc(typeSpec)

							modified = true

						}

					}

				}

			}

		}

		return true

	})



	return modified

}



// generateFunctionDoc generates documentation for a function.

func generateFunctionDoc(funcDecl *ast.FuncDecl) *ast.CommentGroup {

	funcName := funcDecl.Name.Name



	var comment string

	switch {

	case strings.HasPrefix(funcName, "New"):

		comment = fmt.Sprintf("// %s creates a new instance with the provided configuration.", funcName)

	case strings.HasPrefix(funcName, "Get"):

		comment = fmt.Sprintf("// %s retrieves the requested resource.", funcName)

	case strings.HasPrefix(funcName, "Set"):

		comment = fmt.Sprintf("// %s sets the specified value.", funcName)

	case strings.HasPrefix(funcName, "Create"):

		comment = fmt.Sprintf("// %s creates a new resource.", funcName)

	case strings.HasPrefix(funcName, "Update"):

		comment = fmt.Sprintf("// %s updates an existing resource.", funcName)

	case strings.HasPrefix(funcName, "Delete"):

		comment = fmt.Sprintf("// %s removes the specified resource.", funcName)

	case strings.HasPrefix(funcName, "Validate"):

		comment = fmt.Sprintf("// %s validates the input and returns an error if invalid.", funcName)

	case strings.HasPrefix(funcName, "Process"):

		comment = fmt.Sprintf("// %s processes the input and returns the result.", funcName)

	case strings.HasPrefix(funcName, "Handle"):

		comment = fmt.Sprintf("// %s handles the specified operation.", funcName)

	case funcName == "Close":

		comment = fmt.Sprintf("// %s gracefully shuts down and releases resources.", funcName)

	default:

		comment = fmt.Sprintf("// %s performs the requested operation.", funcName)

	}



	return &ast.CommentGroup{

		List: []*ast.Comment{

			{Text: comment},

		},

	}

}



// generateTypeDoc generates documentation for a type.

func generateTypeDoc(typeSpec *ast.TypeSpec) *ast.CommentGroup {

	typeName := typeSpec.Name.Name



	var comment string

	switch {

	case strings.HasSuffix(typeName, "Manager"):

		comment = fmt.Sprintf("// %s manages resources and their lifecycle.", typeName)

	case strings.HasSuffix(typeName, "Factory"):

		comment = fmt.Sprintf("// %s creates and configures instances.", typeName)

	case strings.HasSuffix(typeName, "Client"):

		comment = fmt.Sprintf("// %s provides client functionality for service communication.", typeName)

	case strings.HasSuffix(typeName, "Controller"):

		comment = fmt.Sprintf("// %s implements Kubernetes controller logic.", typeName)

	case strings.HasSuffix(typeName, "Config"):

		comment = fmt.Sprintf("// %s holds configuration settings.", typeName)

	case strings.HasSuffix(typeName, "Handler"):

		comment = fmt.Sprintf("// %s processes requests and generates responses.", typeName)

	case strings.HasSuffix(typeName, "Error"):

		comment = fmt.Sprintf("// %s represents an error condition.", typeName)

	default:

		comment = fmt.Sprintf("// %s represents a core data structure.", typeName)

	}



	return &ast.CommentGroup{

		List: []*ast.Comment{

			{Text: comment},

		},

	}

}



// fixErrorWrapping fixes error handling to use %w format.

func (lf *LintFixer) fixErrorWrapping(file *ast.File) bool {

	modified := false



	ast.Inspect(file, func(n ast.Node) bool {

		if callExpr, ok := n.(*ast.CallExpr); ok {

			if ident, ok := callExpr.Fun.(*ast.SelectorExpr); ok {

				if x, ok := ident.X.(*ast.Ident); ok && x.Name == "fmt" && ident.Sel.Name == "Errorf" {

					// Check if using %s or %v with .Error() method.

					if len(callExpr.Args) >= 2 {

						if formatLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {

							if strings.Contains(formatLit.Value, "%s") || strings.Contains(formatLit.Value, "%v") {

								// Check if second argument is err.Error().

								if callExpr2, ok := callExpr.Args[1].(*ast.CallExpr); ok {

									if selExpr, ok := callExpr2.Fun.(*ast.SelectorExpr); ok && selExpr.Sel.Name == "Error" {

										// Replace %s or %v with %w and remove .Error().

										newFormat := strings.ReplaceAll(formatLit.Value, "%s", "%w")

										newFormat = strings.ReplaceAll(newFormat, "%v", "%w")

										formatLit.Value = newFormat

										callExpr.Args[1] = selExpr.X // Remove .Error() call

										modified = true

									}

								}

							}

						}

					}

				}

			}

		}

		return true

	})



	return modified

}



// fixUnusedParams replaces unused parameters with underscore.

func (lf *LintFixer) fixUnusedParams(file *ast.File) bool {

	modified := false



	ast.Inspect(file, func(n ast.Node) bool {

		if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Body != nil {

			// Find unused parameters.

			if funcDecl.Type.Params != nil {

				for _, field := range funcDecl.Type.Params.List {

					for _, name := range field.Names {

						if name.Name != "_" && !lf.isParamUsed(funcDecl.Body, name.Name) {

							name.Name = "_"

							modified = true

						}

					}

				}

			}

		}

		return true

	})



	return modified

}



// isParamUsed checks if a parameter is used in the function body.

func (lf *LintFixer) isParamUsed(body *ast.BlockStmt, paramName string) bool {

	used := false

	ast.Inspect(body, func(n ast.Node) bool {

		if ident, ok := n.(*ast.Ident); ok && ident.Name == paramName {

			used = true

			return false

		}

		return true

	})

	return used

}



// ensureContextFirst ensures context.Context is the first parameter.

func (lf *LintFixer) ensureContextFirst(file *ast.File) bool {

	modified := false



	ast.Inspect(file, func(n ast.Node) bool {

		if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Type.Params != nil {

			params := funcDecl.Type.Params.List

			if len(params) > 1 {

				// Check if any parameter is context.Context but not first.

				for i, field := range params {

					if i == 0 {

						continue

					}

					if lf.isContextType(field.Type) {

						// Move context parameter to first position.

						contextParam := params[i]

						copy(params[1:i+1], params[0:i])

						params[0] = contextParam

						modified = true

						break

					}

				}

			}

		}

		return true

	})



	return modified

}



// isContextType checks if a type is context.Context.

func (lf *LintFixer) isContextType(expr ast.Expr) bool {

	if selExpr, ok := expr.(*ast.SelectorExpr); ok {

		if ident, ok := selExpr.X.(*ast.Ident); ok {

			return ident.Name == "context" && selExpr.Sel.Name == "Context"

		}

	}

	return false

}



// ApplyCommonFixes applies the most common fixes to a directory.

func ApplyCommonFixes(dir string) error {

	fixer := NewLintFixer()



	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		if err != nil {

			return err

		}



		// Skip non-Go files and generated files.

		if !strings.HasSuffix(path, ".go") {

			return nil

		}



		// Skip generated files.

		if strings.Contains(path, "generated") ||

			strings.Contains(path, "deepcopy") ||

			strings.Contains(path, ".pb.go") {

			return nil

		}



		fmt.Printf("Processing %s\n", path)

		return fixer.FixFile(path)

	})

}



// Quick fixes for specific issues.

var QuickFixes = map[string]string{

	// Package documentation.

	"ST1000": `// Add package comment:

// Package <name> provides <description>.


package <name>`,



	// Exported function documentation.

	"missing-doc-exported": `// Add function documentation:

// FunctionName performs the specified operation.

// Returns an error if the operation fails.

func FunctionName()`,



	// Error wrapping.

	"errorlint": `// Fix error wrapping:

// Bad:  fmt.Errorf("failed: %s", err.Error())

// Good: fmt.Errorf("failed: %w", err)`,



	// Unused variables.

	"ineffassign": `// Fix unused assignment:

// Bad:  result := someFunction()

// Good: _ = someFunction() // or handle the result`,



	// Context patterns.

	"context-first": `// Context should be first parameter:

// Bad:  func Process(data []byte, ctx context.Context)

// Good: func Process(ctx context.Context, data []byte)`,



	// Interface compliance.

	"interface-check": `// Add interface compliance check:

var _ InterfaceName = (*StructName)(nil)`,

}



// PrintQuickFix shows a quick fix for a specific linter error.

func PrintQuickFix(linterName string) {

	if fix, exists := QuickFixes[linterName]; exists {

		fmt.Println(fix)

	} else {

		fmt.Printf("No quick fix available for: %s\n", linterName)

	}

}



func main() {

	if len(os.Args) < 2 {

		fmt.Println("Usage:")

		fmt.Println("  lint-fixes-2025 <directory>  - Apply fixes to directory")

		fmt.Println("  lint-fixes-2025 help <name>  - Show quick fix for linter")

		fmt.Println()

		fmt.Println("Available quick fixes:")

		for name := range QuickFixes {

			fmt.Printf("  %s\n", name)

		}

		return

	}



	command := os.Args[1]



	switch command {

	case "help":

		if len(os.Args) >= 3 {

			PrintQuickFix(os.Args[2])

		} else {

			fmt.Println("Available quick fixes:")

			for name := range QuickFixes {

				fmt.Printf("  %s\n", name)

			}

		}

	default:

		// Treat as directory path.

		if err := ApplyCommonFixes(command); err != nil {

			log.Fatalf("Failed to apply fixes: %v", err)

		}

		fmt.Println("Fixes applied successfully!")

	}

}



// Additional utility functions for manual fixes.



// ReadFileLines reads a file and returns lines for manual processing.

func ReadFileLines(filename string) ([]string, error) {

	file, err := os.Open(filename)

	if err != nil {

		return nil, err

	}

	defer file.Close()



	var lines []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		lines = append(lines, scanner.Text())

	}

	return lines, scanner.Err()

}



// WriteFileLines writes lines back to a file.

func WriteFileLines(filename string, lines []string) error {

	file, err := os.Create(filename)

	if err != nil {

		return err

	}

	defer file.Close()



	writer := bufio.NewWriter(file)

	defer writer.Flush()



	for _, line := range lines {

		if _, err := writer.WriteString(line + "\n"); err != nil {

			return err

		}

	}

	return nil

}



// Common regex patterns for manual fixes.

var LintPatterns = struct {

	UnusedVar        *regexp.Regexp

	MissingErrorWrap *regexp.Regexp

	MissingDoc       *regexp.Regexp

}{

	UnusedVar:        regexp.MustCompile(`^(\s*)(\w+)\s*:=.*$`),

	MissingErrorWrap: regexp.MustCompile(`fmt\.Errorf\([^,]+,\s*.*\.Error\(\)\)`),

	MissingDoc:       regexp.MustCompile(`^func\s+[A-Z]\w*`),

}



// FixUnusedVariables replaces unused variables with underscore.

func FixUnusedVariables(lines, unusedVars []string) []string {

	varMap := make(map[string]bool)

	for _, v := range unusedVars {

		varMap[v] = true

	}



	for i, line := range lines {

		for varName := range varMap {

			if strings.Contains(line, varName+" :=") {

				lines[i] = strings.Replace(line, varName+" :=", "_ :=", 1)

			}

		}

	}

	return lines

}

