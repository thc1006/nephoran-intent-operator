package errors

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	// sourceCodeCache caches source code lines to avoid repeated file reads.
	sourceCodeCache = &sync.Map{}

	// stackTracePool reuses byte slices for stack traces to reduce allocations.
	stackTracePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}
)

// StackTraceOptions configures stack trace capture behavior.
type StackTraceOptions struct {
	// MaxDepth limits the number of stack frames captured.
	MaxDepth int

	// IncludeSource includes source code context in stack frames.
	IncludeSource bool

	// SourceContext specifies how many lines of context to include around the error line.
	SourceContext int

	// SkipFrames specifies how many stack frames to skip at the top.
	SkipFrames int

	// FilterPackages only includes frames from packages matching these prefixes.
	FilterPackages []string

	// ExcludePackages excludes frames from packages matching these prefixes.
	ExcludePackages []string

	// IncludeRuntime includes Go runtime frames in the stack trace.
	IncludeRuntime bool

	// SimplifyPaths simplifies file paths by removing GOPATH/GOROOT prefixes.
	SimplifyPaths bool
}

// DefaultStackTraceOptions returns sensible defaults for stack trace capture.
func DefaultStackTraceOptions() *StackTraceOptions {
	return &StackTraceOptions{
		MaxDepth:      15,
		IncludeSource: true,
		SourceContext: 2,
		SkipFrames:    2, // Skip CaptureStackTrace and the error creation function
		ExcludePackages: []string{
			"runtime",
			"testing",
		},
		IncludeRuntime: false,
		SimplifyPaths:  true,
	}
}

// CaptureStackTrace captures an enhanced stack trace with optional source code context.
func CaptureStackTrace(opts *StackTraceOptions) []StackFrame {
	if opts == nil {
		opts = DefaultStackTraceOptions()
	}

	// Use pooled buffer for stack capture.
	buf := stackTracePool.Get().([]byte)
	defer stackTracePool.Put(buf)

	// Capture the full stack trace.
	n := runtime.Stack(buf, false)
	stackData := buf[:n]

	return parseStackTrace(stackData, opts)
}

// CaptureStackTraceForError captures a stack trace specifically for error creation.
func CaptureStackTraceForError(skip int) []StackFrame {
	opts := DefaultStackTraceOptions()
	opts.SkipFrames = skip + 1 // Add 1 for this function call
	return CaptureStackTrace(opts)
}

// parseStackTrace parses raw stack trace data into structured frames.
func parseStackTrace(stackData []byte, opts *StackTraceOptions) []StackFrame {
	lines := strings.Split(string(stackData), "\n")
	frames := make([]StackFrame, 0, opts.MaxDepth)

	// Skip the first line which is "goroutine X [running]:".
	lineIndex := 1
	frameCount := 0
	skipped := 0

	for lineIndex < len(lines)-1 && frameCount < opts.MaxDepth {
		if lineIndex >= len(lines) {
			break
		}

		// Function line (e.g., "main.main()").
		funcLine := strings.TrimSpace(lines[lineIndex])
		if funcLine == "" {
			lineIndex++
			continue
		}

		lineIndex++
		if lineIndex >= len(lines) {
			break
		}

		// File:line line (e.g., "\t/path/to/file.go:42 +0x123").
		fileLine := strings.TrimSpace(lines[lineIndex])
		if fileLine == "" {
			lineIndex++
			continue
		}

		// Parse file path and line number.
		parts := strings.Split(fileLine, " ")
		if len(parts) < 1 {
			lineIndex++
			continue
		}

		fileAndLine := parts[0]
		lastColon := strings.LastIndex(fileAndLine, ":")
		if lastColon == -1 {
			lineIndex++
			continue
		}

		file := fileAndLine[:lastColon]
		lineNumStr := fileAndLine[lastColon+1:]
		lineNum, err := strconv.Atoi(lineNumStr)
		if err != nil {
			lineIndex++
			continue
		}

		// Create frame.
		frame := StackFrame{
			File:     file,
			Line:     lineNum,
			Function: funcLine,
			Package:  extractPackageName(funcLine),
		}

		// Apply filtering.
		if shouldSkipFrame(&frame, opts) {
			if skipped < opts.SkipFrames {
				skipped++
			}
			lineIndex++
			continue
		}

		// Skip if we haven't reached the skip threshold yet.
		if skipped < opts.SkipFrames {
			skipped++
			lineIndex++
			continue
		}

		// Simplify paths if requested.
		if opts.SimplifyPaths {
			frame.File = simplifyPath(frame.File)
		}

		// Add source code context if requested.
		if opts.IncludeSource {
			frame.Source = getSourceContext(frame.File, frame.Line, opts.SourceContext)
		}

		frames = append(frames, frame)
		frameCount++
		lineIndex++
	}

	return frames
}

// shouldSkipFrame determines if a frame should be skipped based on filtering options.
func shouldSkipFrame(frame *StackFrame, opts *StackTraceOptions) bool {
	// Skip runtime frames if not included.
	if !opts.IncludeRuntime && strings.HasPrefix(frame.Package, "runtime") {
		return true
	}

	// Check exclude packages.
	for _, exclude := range opts.ExcludePackages {
		if strings.HasPrefix(frame.Package, exclude) ||
			strings.Contains(frame.Function, exclude) {
			return true
		}
	}

	// Check filter packages (if specified, only include matching packages).
	if len(opts.FilterPackages) > 0 {
		found := false
		for _, filter := range opts.FilterPackages {
			if strings.HasPrefix(frame.Package, filter) ||
				strings.Contains(frame.Function, filter) {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	return false
}

// extractPackageName extracts the package name from a function signature.
func extractPackageName(funcName string) string {
	// Remove receiver types like "(*Type).Method" -> "package.(*Type).Method".
	if idx := strings.LastIndex(funcName, "."); idx != -1 {
		packagePart := funcName[:idx]

		// Handle method calls with receivers.
		if strings.Contains(packagePart, "(") {
			if parenIdx := strings.Index(packagePart, "("); parenIdx != -1 {
				packagePart = packagePart[:parenIdx]
			}
		}

		// Handle nested packages.
		if lastSlash := strings.LastIndex(packagePart, "/"); lastSlash != -1 {
			return packagePart[lastSlash+1:]
		}

		return packagePart
	}

	return "main"
}

// simplifyPath removes GOPATH/GOROOT prefixes from file paths.
func simplifyPath(path string) string {
	// Try to remove GOROOT.
	if goroot := runtime.GOROOT(); goroot != "" {
		if rel, err := filepath.Rel(goroot, path); err == nil && !strings.HasPrefix(rel, "..") {
			return "$GOROOT/" + rel
		}
	}

	// Try to remove GOPATH.
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		for _, gp := range filepath.SplitList(gopath) {
			if rel, err := filepath.Rel(filepath.Join(gp, "src"), path); err == nil && !strings.HasPrefix(rel, "..") {
				return rel
			}
		}
	}

	// Try to remove module path for Go modules.
	if rel := getRelativeToModule(path); rel != "" {
		return rel
	}

	return path
}

// getRelativeToModule attempts to get the path relative to the Go module root.
func getRelativeToModule(path string) string {
	// Walk up the directory tree looking for go.mod.
	dir := filepath.Dir(path)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if rel, err := filepath.Rel(dir, path); err == nil {
				return rel
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return ""
}

// getSourceContext reads source code context around the specified line.
func getSourceContext(file string, line, context int) string {
	if context <= 0 {
		return ""
	}

	// Check cache first.
	cacheKey := fmt.Sprintf("%s:%d:%d", file, line, context)
	if cached, ok := sourceCodeCache.Load(cacheKey); ok {
		return cached.(string)
	}

	source := readSourceContext(file, line, context)

	// Cache the result.
	sourceCodeCache.Store(cacheKey, source)

	return source
}

// readSourceContext reads the actual source code context.
func readSourceContext(file string, line, context int) string {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Sprintf("// Could not read source: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	currentLine := 1
	startLine := line - context
	endLine := line + context

	if startLine < 1 {
		startLine = 1
	}

	var lines []string
	for scanner.Scan() {
		if currentLine >= startLine && currentLine <= endLine {
			prefix := "  "
			if currentLine == line {
				prefix = "→ " // Mark the actual error line
			}
			lines = append(lines, fmt.Sprintf("%s%d: %s", prefix, currentLine, scanner.Text()))
		}

		if currentLine > endLine {
			break
		}

		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("// Error reading source: %v", err)
	}

	return strings.Join(lines, "\n")
}

// FormatStackTrace formats a stack trace for human-readable output.
func FormatStackTrace(frames []StackFrame, includeSource bool) string {
	var builder strings.Builder

	for i, frame := range frames {
		builder.WriteString(fmt.Sprintf("#%d %s\n", i, frame.Function))
		builder.WriteString(fmt.Sprintf("    at %s:%d\n", frame.File, frame.Line))

		if includeSource && frame.Source != "" {
			// Indent source code.
			sourceLines := strings.Split(frame.Source, "\n")
			for _, sourceLine := range sourceLines {
				if strings.TrimSpace(sourceLine) != "" {
					builder.WriteString(fmt.Sprintf("    %s\n", sourceLine))
				}
			}
		}

		if i < len(frames)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// FilterStackTrace applies post-capture filtering to a stack trace.
func FilterStackTrace(frames []StackFrame, predicate func(StackFrame) bool) []StackFrame {
	filtered := make([]StackFrame, 0, len(frames))
	for _, frame := range frames {
		if predicate(frame) {
			filtered = append(filtered, frame)
		}
	}
	return filtered
}

// getSafeFunctionName safely extracts function name with proper error handling.
func getSafeFunctionName(fn *runtime.Func) string {
	if fn == nil {
		return ""
	}

	// Use defer/recover to catch any potential panics from fn.Name().
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't propagate it - return empty string instead.
			// This prevents the entire error handling system from crashing.
		}
	}()

	// Even though fn is not nil, fn.Name() can still panic in edge cases.
	// where the PC doesn't correspond to a valid function entry point.
	name := fn.Name()
	if name == "" {
		return "<unnamed>"
	}

	return name
}

// GetCallerInfo returns information about the caller at the specified skip level.
func GetCallerInfo(skip int) (file string, line int, function string, ok bool) {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "", 0, "", false
	}

	fn := runtime.FuncForPC(pc)
	function = getSafeFunctionName(fn)
	if function == "" {
		function = "<unknown>"
	}

	return file, line, function, true
}

// GetCurrentStackDepth returns the current stack depth.
func GetCurrentStackDepth() int {
	depth := 0
	for {
		_, _, _, ok := runtime.Caller(depth + 1)
		if !ok {
			break
		}
		depth++
	}
	return depth
}

// IsInPackage checks if the current call stack includes frames from the specified package.
func IsInPackage(packageName string, maxDepth int) bool {
	for i := 1; i <= maxDepth; i++ {
		pc, _, _, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		funcName := getSafeFunctionName(fn)
		if funcName != "" && strings.Contains(funcName, packageName) {
			return true
		}
	}

	return false
}

// StackFrameFilter is a predicate function for filtering stack frames.
type StackFrameFilter func(StackFrame) bool

// Common stack frame filters.
var (
	// SkipRuntimeFrames filters out Go runtime frames.
	SkipRuntimeFrames StackFrameFilter = func(frame StackFrame) bool {
		return !strings.HasPrefix(frame.Package, "runtime")
	}

	// SkipTestingFrames filters out testing framework frames.
	SkipTestingFrames StackFrameFilter = func(frame StackFrame) bool {
		return !strings.Contains(frame.Function, "testing") &&
			!strings.Contains(frame.Function, "ginkgo") &&
			!strings.Contains(frame.Function, "gomega")
	}

	// OnlyApplicationFrames keeps only application frames (non-stdlib).
	OnlyApplicationFrames StackFrameFilter = func(frame StackFrame) bool {
		return !strings.Contains(frame.File, "$GOROOT/") &&
			!strings.HasPrefix(frame.Package, "runtime") &&
			!strings.HasPrefix(frame.Package, "net/") &&
			!strings.HasPrefix(frame.Package, "os/") &&
			!strings.HasPrefix(frame.Package, "io/") &&
			!strings.HasPrefix(frame.Package, "fmt") &&
			!strings.HasPrefix(frame.Package, "log")
	}
)

// CombineFilters combines multiple filters with AND logic.
func CombineFilters(filters ...StackFrameFilter) StackFrameFilter {
	return func(frame StackFrame) bool {
		for _, filter := range filters {
			if !filter(frame) {
				return false
			}
		}
		return true
	}
}

// StackTraceAnalyzer provides analysis capabilities for stack traces.
type StackTraceAnalyzer struct {
	packageStats  map[string]int
	functionStats map[string]int
	fileStats     map[string]int
}

// NewStackTraceAnalyzer creates a new stack trace analyzer.
func NewStackTraceAnalyzer() *StackTraceAnalyzer {
	return &StackTraceAnalyzer{
		packageStats:  make(map[string]int),
		functionStats: make(map[string]int),
		fileStats:     make(map[string]int),
	}
}

// Analyze analyzes a stack trace and collects statistics.
func (sta *StackTraceAnalyzer) Analyze(frames []StackFrame) {
	for _, frame := range frames {
		sta.packageStats[frame.Package]++
		sta.functionStats[frame.Function]++
		sta.fileStats[frame.File]++
	}
}

// GetPackageStats returns statistics about package usage in stack traces.
func (sta *StackTraceAnalyzer) GetPackageStats() map[string]int {
	return sta.packageStats
}

// GetFunctionStats returns statistics about function usage in stack traces.
func (sta *StackTraceAnalyzer) GetFunctionStats() map[string]int {
	return sta.functionStats
}

// GetFileStats returns statistics about file usage in stack traces.
func (sta *StackTraceAnalyzer) GetFileStats() map[string]int {
	return sta.fileStats
}

// Reset clears all collected statistics.
func (sta *StackTraceAnalyzer) Reset() {
	sta.packageStats = make(map[string]int)
	sta.functionStats = make(map[string]int)
	sta.fileStats = make(map[string]int)
}
